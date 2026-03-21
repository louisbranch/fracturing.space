package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestApplyRollOutcome_PartialRetrySkipsRepeatedGMFearSet(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := testTimestamp

	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     1,
	}

	rollEvent := fearRollEvent(t, "req-roll-partial-retry").appendTo(eventStore)

	_, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-partial-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"before":0,"after":1}`),
	})
	if err != nil {
		t.Fatalf("append gm fear event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-partial-retry"},
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
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "outcome",
					EntityID:    "req-roll-partial-retry",
					PayloadJSON: []byte(`{"request_id":"req-roll-partial-retry","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
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
					RequestID:   "req-roll-partial-retry",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-partial-retry")
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
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	for _, effect := range payload.PreEffects {
		if effect.Type == "sys.daggerheart.gm_fear_changed" {
			t.Fatal("did not expect gm fear pre_effect on partial retry")
		}
	}
	if snap := dhStore.Snapshots["camp-1"]; snap.GMFear != 1 {
		t.Fatalf("gm fear = %d, want %d", snap.GMFear, 1)
	}
}

func TestApplyRollOutcome_PartialRetrySkipsRepeatedCharacterPatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	rollEvent := newRollEvent(t, "req-roll-patch-retry").appendTo(eventStore)

	_, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-patch-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","hope_before":2,"hope_after":3,"stress_before":3,"stress_after":3}`),
	})
	if err != nil {
		t.Fatalf("append patch event: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-patch-retry",
				EntityType:  "outcome",
				EntityID:    "req-roll-patch-retry",
				PayloadJSON: []byte(`{"request_id":"req-roll-patch-retry"}`),
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-patch-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if resp.RollSeq != rollEvent.Seq {
		t.Fatalf("roll seq = %d, want %d", resp.RollSeq, rollEvent.Seq)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("action.outcome.apply") {
		t.Fatalf("expected only outcome apply command, got %+v", domain.commands)
	}
}
