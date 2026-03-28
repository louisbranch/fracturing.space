package daggerheart

import (
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

func mustStandardConditionState(t *testing.T, code string) rules.ConditionState {
	t.Helper()
	state, err := rules.StandardConditionState(code)
	if err != nil {
		t.Fatalf("standard condition state %q: %v", code, err)
	}
	return state
}

func protoStandardConditionState(condition pb.DaggerheartCondition) *pb.DaggerheartConditionState {
	code := ""
	switch condition {
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
		code = rules.ConditionHidden
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
		code = rules.ConditionRestrained
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
		code = rules.ConditionVulnerable
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED:
		code = rules.ConditionCloaked
	}
	return &pb.DaggerheartConditionState{
		Id:       code,
		Code:     code,
		Label:    code,
		Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
		Standard: condition,
	}
}

func projectionStandardConditionState(code string) projectionstore.DaggerheartConditionState {
	return projectionstore.DaggerheartConditionState{
		ID:       code,
		Class:    "standard",
		Standard: code,
		Code:     code,
		Label:    code,
	}
}

// --- ApplyConditions gap fills ---

func TestApplyConditions_AddCondition_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: "char-1",
		Conditions:  []rules.ConditionState{mustStandardConditionState(t, rules.ConditionHidden)},
		Added:       []rules.ConditionState{mustStandardConditionState(t, rules.ConditionHidden)},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-add",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-add")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:    "camp-1",
		CharacterId:   "char-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.AddedConditions) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyConditions_RemoveCondition_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []projectionstore.DaggerheartConditionState{
		projectionStandardConditionState(rules.ConditionHidden),
		projectionStandardConditionState(rules.ConditionVulnerable),
	}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: "char-1",
		Conditions:  []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Removed:     []rules.ConditionState{mustStandardConditionState(t, rules.ConditionHidden)},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-remove",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-remove")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:         "camp-1",
		CharacterId:        "char-1",
		RemoveConditionIds: []string{rules.ConditionHidden},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.RemovedConditions) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_AddAndRemove(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []projectionstore.DaggerheartConditionState{
		projectionStandardConditionState(rules.ConditionHidden),
	}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheartpayload.ConditionChangedPayload{
		CharacterID: "char-1",
		Conditions:  []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Added:       []rules.ConditionState{mustStandardConditionState(t, rules.ConditionVulnerable)},
		Removed:     []rules.ConditionState{mustStandardConditionState(t, rules.ConditionHidden)},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-both",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-both")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:         "camp-1",
		CharacterId:        "char-1",
		AddConditions:      []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
		RemoveConditionIds: []string{rules.ConditionHidden},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.AddedConditions) == 0 {
		t.Fatal("expected added conditions")
	}
	if len(resp.RemovedConditions) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:         "camp-1",
		CharacterId:        "char-1",
		AddConditions:      []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
		RemoveConditionIds: []string{rules.ConditionHidden},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyGmMove gap fills ---
