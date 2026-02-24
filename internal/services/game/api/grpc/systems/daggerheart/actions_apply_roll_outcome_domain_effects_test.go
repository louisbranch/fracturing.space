package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestApplyRollOutcome_UsesDomainEngineForGmConsequenceGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-1"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"before":0,"after":1}`),
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "outcome",
					EntityID:    "req-roll-1",
					PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(payload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(payload.PreEffects))
	}
	if len(payload.PostEffects) != 2 {
		t.Fatalf("post_effects length = %d, want 2", len(payload.PostEffects))
	}
	if payload.PostEffects[0].Type != "session.gate_opened" {
		t.Fatalf("post_effects[0].type = %s, want %s", payload.PostEffects[0].Type, "session.gate_opened")
	}
	if payload.PostEffects[1].Type != "session.spotlight_set" {
		t.Fatalf("post_effects[1].type = %s, want %s", payload.PostEffects[1].Type, "session.spotlight_set")
	}
}

func TestApplyRollOutcome_CorrelationIDOnIntermediateCommands(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-corr",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-corr",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-corr",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-corr"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-corr",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"before":0,"after":1}`),
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-corr",
					EntityType:  "outcome",
					EntityID:    "req-roll-corr",
					PayloadJSON: []byte(`{"request_id":"req-roll-corr","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-corr",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-corr",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-corr")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if len(domain.commands) < 2 {
		t.Fatalf("expected at least 2 domain commands, got %d", len(domain.commands))
	}
	// All intermediate commands (before the final outcome.apply) must carry the
	// roll's request_id as correlation_id so consequence events form a queryable set.
	for i, cmd := range domain.commands {
		if cmd.CorrelationID != "req-roll-corr" {
			t.Errorf("command[%d] (%s) CorrelationID = %q, want %q",
				i, cmd.Type, cmd.CorrelationID, "req-roll-corr")
		}
	}
}

func TestApplyRollOutcome_UsesDomainEngineForCharacterStatePatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := state.Stress
	stressAfter := stressBefore

	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var got action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &got); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(got.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(got.PreEffects))
	}
	var foundPatchEvent bool
	var foundOutcomeEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		switch evt.Type {
		case event.Type("sys.daggerheart.character_state_patched"):
			foundPatchEvent = true
		case event.Type("action.outcome_applied"):
			foundOutcomeEvent = true
		}
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
	if !foundOutcomeEvent {
		t.Fatal("expected outcome applied event")
	}
}

func TestApplyRollOutcome_UsesDomainEngineForConditionChange(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	state.Stress = profile.StressMax
	state.Conditions = []string{daggerheart.ConditionVulnerable}
	dhStore.States["camp-1:char-1"] = state
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_CRITICAL_SUCCESS.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"crit":         true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := profile.StressMax
	stressAfter := stressBefore - 1
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	rollSeq := rollEvent.Seq
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionVulnerable},
		ConditionsAfter:  []string{},
		Removed:          []string{daggerheart.ConditionVulnerable},
		RollSeq:          &rollSeq,
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 3 {
		t.Fatalf("expected domain to be called three times, got %d", domain.calls)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("sys.daggerheart.condition.change") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "sys.daggerheart.condition.change")
	}
	if domain.commands[2].Type != command.Type("action.outcome.apply") {
		t.Fatalf("third command type = %s, want %s", domain.commands[2].Type, "action.outcome.apply")
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[2].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(payload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(payload.PreEffects))
	}
	var foundConditionEvent bool
	var foundPatchEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		switch evt.Type {
		case event.Type("sys.daggerheart.condition_changed"):
			foundConditionEvent = true
		case event.Type("sys.daggerheart.character_state_patched"):
			foundPatchEvent = true
		}
	}
	if !foundConditionEvent {
		t.Fatal("expected condition changed event")
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
}
