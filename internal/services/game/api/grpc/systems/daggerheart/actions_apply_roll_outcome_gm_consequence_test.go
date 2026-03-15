package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
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
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
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
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-1")
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
