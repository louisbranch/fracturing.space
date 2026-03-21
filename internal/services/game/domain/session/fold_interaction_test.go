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
		{typ: EventTypeSceneActivated, payload: SceneActivatedPayload{ActiveSceneID: "scene-1"}},
		{typ: EventTypeGMAuthoritySet, payload: GMAuthoritySetPayload{ParticipantID: "gm-1"}},
		{typ: EventTypeOOCOpened, payload: OOCOpenedPayload{
			Reason:                   "rules",
			RequestedByParticipantID: "gm-1",
			InterruptedSceneID:       "scene-1",
			InterruptedPhaseID:       "phase-1",
			InterruptedPhaseStatus:   "GM_REVIEW",
		}},
		{typ: EventTypeOOCPosted, payload: OOCPostedPayload{PostID: "ooc-1", ParticipantID: "p1", Body: "question"}},
		{typ: EventTypeOOCReadyMarked, payload: OOCReadyMarkedPayload{ParticipantID: "p1"}},
		{typ: EventTypeOOCReadyCleared, payload: OOCReadyClearedPayload{ParticipantID: "p1"}},
		{typ: EventTypeOOCClosed, payload: OOCClosedPayload{Reason: "resume"}},
		{typ: EventTypeOOCResolved, payload: OOCResolvedPayload{Resolution: "resume_original_phase"}},
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
	if state.OOCRequestedByParticipantID != "" || state.OOCReason != "" || state.OOCInterruptedSceneID != "" || state.OOCInterruptedPhaseID != "" || state.OOCInterruptedPhaseStatus != "" || state.OOCResolutionPending {
		t.Fatalf("ooc interruption state = %#v, want cleared", state)
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
		EventTypeSceneActivated,
		EventTypeGMAuthoritySet,
		EventTypeOOCOpened,
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

func TestFold_OOCClosedSetsResolutionPendingWhenInterruptionExists(t *testing.T) {
	t.Parallel()

	next, err := Fold(State{
		OOCPaused:             true,
		OOCInterruptedSceneID: "scene-1",
		OOCInterruptedPhaseID: "phase-1",
		OOCReadyParticipants:  map[ids.ParticipantID]bool{"p1": true},
	}, event.Event{Type: EventTypeOOCClosed})
	if err != nil {
		t.Fatalf("fold resume: %v", err)
	}
	if next.OOCPaused {
		t.Fatal("expected ooc pause to clear")
	}
	if !next.OOCResolutionPending {
		t.Fatal("expected resolution pending to be set")
	}
	if next.OOCReadyParticipants != nil {
		t.Fatalf("ready participants = %#v, want nil", next.OOCReadyParticipants)
	}
}
