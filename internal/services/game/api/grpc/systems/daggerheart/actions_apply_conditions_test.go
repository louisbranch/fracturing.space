package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyConditions gap fills ---

func TestApplyConditions_AddCondition_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionHidden},
		Added:            []string{daggerheart.ConditionHidden},
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-add")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyConditions_RemoveCondition_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"hidden", "vulnerable"}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionHidden, daggerheart.ConditionVulnerable},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Removed:          []string{daggerheart.ConditionHidden},
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-remove")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_AddAndRemove(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"hidden"}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionHidden},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Added:            []string{daggerheart.ConditionVulnerable},
		Removed:          []string{daggerheart.ConditionHidden},
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-both")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyGmMove gap fills ---
