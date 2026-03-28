package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
)

// --- ApplyAdversaryConditions tests ---

func TestApplyAdversaryConditions_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryConditions_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		AdversaryId: "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{
			protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE),
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "camp-1",
		AddConditions: []*pb.DaggerheartConditionState{
			protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE),
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:    "camp-1",
		AdversaryId:   "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_NoConditions(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:         "camp-1",
		AdversaryId:        "adv-1",
		AddConditions:      []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
		RemoveConditionIds: []string{rules.ConditionVulnerable},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_AddCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	payload := daggerheartpayload.AdversaryConditionChangedPayload{
		AdversaryID: "adv-1",
		Conditions:  []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Added:       []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-conditions-add",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adv-conditions-add")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:    "camp-1",
		AdversaryId:   "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if len(resp.AddedConditions) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyAdversaryConditions_RemoveCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition on the adversary.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []projectionstore.DaggerheartConditionState{projectionStandardConditionState(rules.ConditionVulnerable)}
	dhStore.adversaries["camp-1:adv-1"] = adv
	eventStore := svc.stores.Event.(*fakeEventStore)
	payload := daggerheartpayload.AdversaryConditionChangedPayload{
		AdversaryID: "adv-1",
		Conditions:  nil,
		Removed:     []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-conditions-remove",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adv-conditions-remove")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:         "camp-1",
		AdversaryId:        "adv-1",
		RemoveConditionIds: []string{rules.ConditionVulnerable},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if len(resp.RemovedConditions) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyAdversaryConditions_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	payload := daggerheartpayload.AdversaryConditionChangedPayload{
		AdversaryID: "adv-1",
		Conditions:  []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Added:       []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Source:      "test",
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adversary-conditions",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adversary-conditions")
	_, err = svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:    "camp-1",
		AdversaryId:   "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
		Source:        "test",
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.adversary_condition.change") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.adversary_condition.change")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		AdversaryID     string                 `json:"adversary_id"`
		ConditionsAfter []rules.ConditionState `json:"conditions_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode adversary condition command payload: %v", err)
	}
	if got.AdversaryID != "adv-1" {
		t.Fatalf("command adversary id = %s, want %s", got.AdversaryID, "adv-1")
	}
	if len(got.ConditionsAfter) != 1 || got.ConditionsAfter[0].Code != rules.ConditionVulnerable {
		t.Fatalf("command conditions_after = %v, want [%s]", got.ConditionsAfter, rules.ConditionVulnerable)
	}
}

func TestApplyAdversaryConditions_NoChanges(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition that we try to re-add.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []projectionstore.DaggerheartConditionState{projectionStandardConditionState(rules.ConditionVulnerable)}
	dhStore.adversaries["camp-1:adv-1"] = adv

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:    "camp-1",
		AdversaryId:   "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
