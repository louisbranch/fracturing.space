package session

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestFold_SessionInteractionLifecycle(t *testing.T) {
	t.Parallel()

	state := State{SessionID: "sess-1", Started: true}
	events := []struct {
		typ     event.Type
		payload any
	}{
		{typ: EventTypeActiveSceneSet, payload: ActiveSceneSetPayload{ActiveSceneID: "scene-1"}},
		{typ: EventTypeGMAuthoritySet, payload: GMAuthoritySetPayload{ParticipantID: "gm-1"}},
		{typ: EventTypeOOCPaused, payload: OOCPausedPayload{Reason: "rules"}},
		{typ: EventTypeOOCReadyMarked, payload: OOCReadyMarkedPayload{ParticipantID: "p1"}},
		{typ: EventTypeOOCReadyCleared, payload: OOCReadyClearedPayload{ParticipantID: "p1"}},
		{typ: EventTypeOOCResumed, payload: OOCResumedPayload{Reason: "resume"}},
		{typ: EventTypeAITurnQueued, payload: AITurnQueuedPayload{
			TurnToken:          "turn-1",
			OwnerParticipantID: "gm-ai",
			SourceEventType:    "scene.player_phase_ended",
			SourceSceneID:      "scene-1",
			SourcePhaseID:      "phase-1",
		}},
		{typ: EventTypeAITurnRunning, payload: AITurnRunningPayload{TurnToken: "turn-1"}},
		{typ: EventTypeAITurnFailed, payload: AITurnFailedPayload{TurnToken: "turn-1", LastError: "timeout"}},
		{typ: EventTypeAITurnCleared, payload: AITurnClearedPayload{TurnToken: "turn-1", Reason: "retry"}},
	}

	for _, item := range events {
		payloadJSON, err := json.Marshal(item.payload)
		if err != nil {
			t.Fatalf("marshal payload for %s: %v", item.typ, err)
		}
		state, err = Fold(state, event.Event{Type: item.typ, PayloadJSON: payloadJSON})
		if err != nil {
			t.Fatalf("fold %s: %v", item.typ, err)
		}
	}

	if state.ActiveSceneID != ids.SceneID("scene-1") {
		t.Fatalf("active scene = %q", state.ActiveSceneID)
	}
	if state.GMAuthorityParticipantID != ids.ParticipantID("gm-1") {
		t.Fatalf("gm authority = %q", state.GMAuthorityParticipantID)
	}
	if state.OOCPaused {
		t.Fatal("ooc paused = true, want false after resume")
	}
	if state.OOCReadyParticipants != nil {
		t.Fatalf("ready participants = %#v, want nil after resume", state.OOCReadyParticipants)
	}
	if state.AITurnStatus != AITurnStatusIdle || state.AITurnToken != "" || state.AITurnLastError != "" {
		t.Fatalf("ai turn state = %#v", state)
	}
}

func TestFold_SessionInteractionInvalidPayloadsReturnErrors(t *testing.T) {
	t.Parallel()

	corrupt := []byte(`{`)
	for _, evtType := range []event.Type{
		EventTypeActiveSceneSet,
		EventTypeGMAuthoritySet,
		EventTypeOOCReadyMarked,
		EventTypeOOCReadyCleared,
		EventTypeAITurnQueued,
		EventTypeAITurnRunning,
		EventTypeAITurnFailed,
	} {
		if _, err := Fold(State{}, event.Event{Type: evtType, PayloadJSON: corrupt}); err == nil {
			t.Fatalf("expected error for %s", evtType)
		}
	}
}
