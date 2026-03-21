package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
)

func TestApplyConditions_LifeStateOnly(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	after := mechanics.LifeStateUnconscious
	patchJSON := mustCharacterStatePatchedJSON(t, "char-1", after)
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-life",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-life")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS {
		t.Fatalf("life_state = %v, want UNCONSCIOUS", resp.State.LifeState)
	}

	events := eventStore.Events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	last := events[len(events)-1]
	if last.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("last event type = %s, want %s", last.Type, event.Type("sys.daggerheart.character_state_patched"))
	}
}

func TestApplyConditions_LifeStateNoChange(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_InvalidStoredLifeState(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.LifeState = "not-a-life-state"
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_NoConditionChanges(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []projectionstore.DaggerheartConditionState{
		projectionStandardConditionState(rules.ConditionVulnerable),
	}
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:    "camp-1",
		CharacterId:   "char-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:    "camp-1",
		CharacterId:   "char-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	conditionJSON := mustConditionChangedJSON(t, "char-1", []string{rules.ConditionHidden}, []string{rules.ConditionHidden})

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:    "camp-1",
		CharacterId:   "char-1",
		AddConditions: []*pb.DaggerheartConditionState{protoStandardConditionState(pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.condition.change") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.condition.change")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheartpayload.ConditionChangePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode condition command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if len(got.ConditionsAfter) != 1 || got.ConditionsAfter[0].Code != rules.ConditionHidden {
		t.Fatalf("command conditions_after = %v, want %s", got.ConditionsAfter, rules.ConditionHidden)
	}
	var foundConditionEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.condition_changed") {
			foundConditionEvent = true
			break
		}
	}
	if !foundConditionEvent {
		t.Fatal("expected condition changed event")
	}
}

func TestApplyConditions_UsesDomainEngineForLifeState(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	before := daggerheartstate.LifeStateAlive
	after := mechanics.LifeStateUnconscious
	patchJSON := mustCharacterStatePatchedJSON(t, "char-1", after)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheartpayload.CharacterStatePatchPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode patch command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.LifeStateBefore == nil || *got.LifeStateBefore != before {
		t.Fatalf("command life_state_before = %v, want %s", got.LifeStateBefore, before)
	}
	if got.LifeStateAfter == nil || *got.LifeStateAfter != after {
		t.Fatalf("command life_state_after = %v, want %s", got.LifeStateAfter, after)
	}
	var foundStateEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.character_state_patched") {
			foundStateEvent = true
			break
		}
	}
	if !foundStateEvent {
		t.Fatal("expected character state patched event")
	}
}
