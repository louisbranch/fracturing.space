package outcometransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestHandlerBuildApplyRollOutcomeIdempotentResponseIncludesGMFear(t *testing.T) {
	handler, _, _ := newTestHandler()

	resp, err := handler.buildApplyRollOutcomeIdempotentResponse(context.Background(), "camp-1", 7, []string{"char-1"}, true, true)
	if err != nil {
		t.Fatalf("buildApplyRollOutcomeIdempotentResponse returned error: %v", err)
	}
	if got := len(resp.GetUpdated().GetCharacterStates()); got != 1 {
		t.Fatalf("character_states len = %d, want 1", got)
	}
	if got := resp.GetUpdated().GetGmFear(); got != 2 {
		t.Fatalf("gm_fear = %d, want 2", got)
	}
}

func TestHandlerSessionRequestEventExistsMatchesEvent(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{})
	if _, err := events.AppendEvent(context.Background(), event.Event{
		CampaignID: "camp-1",
		Timestamp:  testTimestamp,
		Type:       eventTypeActionOutcomeApplied,
		SessionID:  "sess-1",
		RequestID:  "req-1",
		EntityType: "outcome",
		EntityID:   "req-1",
	}); err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	exists, err := handler.sessionRequestEventExists(context.Background(), "camp-1", "sess-1", roll.Seq, "req-1", eventTypeActionOutcomeApplied, "req-1")
	if err != nil {
		t.Fatalf("sessionRequestEventExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected matching session event to exist")
	}
}
