package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func TestApplyRollOutcome_UsesSystemAndCoreCommandBoundary(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-single-boundary",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
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
		RequestID:   "req-roll-single-boundary",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-single-boundary",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeAfter := hopeBefore + 1
	patchPayload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		Hope:        &hopeAfter,
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
				RequestID:     "req-roll-single-boundary",
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
				RequestID:   "req-roll-single-boundary",
				EntityType:  "outcome",
				EntityID:    "req-roll-single-boundary",
				PayloadJSON: []byte(`{"request_id":"req-roll-single-boundary","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-single-boundary")
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
}

func TestApplyRollOutcome_CorrelationIDOnIntermediateCommands(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-corr",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
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
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-corr")
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
	for i, cmd := range domain.commands {
		if cmd.CorrelationID != "req-roll-corr" {
			t.Errorf("command[%d] (%s) CorrelationID = %q, want %q", i, cmd.Type, cmd.CorrelationID, "req-roll-corr")
		}
	}
}
