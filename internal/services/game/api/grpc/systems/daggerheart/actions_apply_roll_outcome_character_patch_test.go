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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestApplyRollOutcome_UsesDomainEngineForCharacterStatePatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
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
		CharacterID: "char-1",
		Hope:        &hopeAfter,
		Stress:      &stressAfter,
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
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-1")
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
