package projection

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplySessionAITurnQueued(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	err := applier.applySessionAITurnQueued(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.AITurnQueuedPayload{
		TurnToken:          "token-abc",
		OwnerParticipantID: ids.ParticipantID("p-gm"),
		SourceEventType:    "scene.player_phase_accepted",
		SourceSceneID:      ids.SceneID("scene-1"),
		SourcePhaseID:      "phase-1",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.AITurn.Status != session.AITurnStatusQueued {
		t.Fatalf("status = %q, want %q", got.AITurn.Status, session.AITurnStatusQueued)
	}
	if got.AITurn.TurnToken != "token-abc" {
		t.Fatalf("turn token = %q, want %q", got.AITurn.TurnToken, "token-abc")
	}
	if got.AITurn.OwnerParticipantID != "p-gm" {
		t.Fatalf("owner participant id = %q, want %q", got.AITurn.OwnerParticipantID, "p-gm")
	}
	if got.AITurn.SourceEventType != "scene.player_phase_accepted" {
		t.Fatalf("source event type = %q, want %q", got.AITurn.SourceEventType, "scene.player_phase_accepted")
	}
	if got.AITurn.SourceSceneID != "scene-1" {
		t.Fatalf("source scene id = %q, want %q", got.AITurn.SourceSceneID, "scene-1")
	}
	if got.AITurn.SourcePhaseID != "phase-1" {
		t.Fatalf("source phase id = %q, want %q", got.AITurn.SourcePhaseID, "phase-1")
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionAITurnRunning(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	// Seed with a queued AI turn that has a previous error.
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		AITurn: storage.SessionAITurn{
			Status:             session.AITurnStatusQueued,
			TurnToken:          "token-old",
			OwnerParticipantID: "p-gm",
			SourceEventType:    "scene.player_phase_accepted",
			LastError:          "previous failure",
		},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)

	err := applier.applySessionAITurnRunning(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.AITurnRunningPayload{
		TurnToken: "token-old",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.AITurn.Status != session.AITurnStatusRunning {
		t.Fatalf("status = %q, want %q", got.AITurn.Status, session.AITurnStatusRunning)
	}
	if got.AITurn.TurnToken != "token-old" {
		t.Fatalf("turn token = %q, want %q", got.AITurn.TurnToken, "token-old")
	}
	if got.AITurn.LastError != "" {
		t.Fatalf("last error = %q, want empty (cleared on running)", got.AITurn.LastError)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionAITurnFailed(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		AITurn: storage.SessionAITurn{
			Status:    session.AITurnStatusRunning,
			TurnToken: "token-abc",
		},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 12, 2, 0, 0, time.UTC)

	err := applier.applySessionAITurnFailed(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.AITurnFailedPayload{
		TurnToken: "token-abc",
		LastError: "context deadline exceeded",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.AITurn.Status != session.AITurnStatusFailed {
		t.Fatalf("status = %q, want %q", got.AITurn.Status, session.AITurnStatusFailed)
	}
	if got.AITurn.TurnToken != "token-abc" {
		t.Fatalf("turn token = %q, want %q", got.AITurn.TurnToken, "token-abc")
	}
	if got.AITurn.LastError != "context deadline exceeded" {
		t.Fatalf("last error = %q, want %q", got.AITurn.LastError, "context deadline exceeded")
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionAITurnCleared(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		AITurn: storage.SessionAITurn{
			Status:             session.AITurnStatusFailed,
			TurnToken:          "token-abc",
			OwnerParticipantID: "p-gm",
			SourceEventType:    "scene.player_phase_accepted",
			SourceSceneID:      "scene-1",
			SourcePhaseID:      "phase-1",
			LastError:          "something went wrong",
		},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 12, 3, 0, 0, time.UTC)

	err := applier.applySessionAITurnCleared(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.AITurnClearedPayload{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.AITurn.Status != session.AITurnStatusIdle {
		t.Fatalf("status = %q, want %q", got.AITurn.Status, session.AITurnStatusIdle)
	}
	if got.AITurn.TurnToken != "" {
		t.Fatalf("turn token = %q, want empty", got.AITurn.TurnToken)
	}
	if got.AITurn.OwnerParticipantID != "" {
		t.Fatalf("owner participant id = %q, want empty", got.AITurn.OwnerParticipantID)
	}
	if got.AITurn.SourceEventType != "" {
		t.Fatalf("source event type = %q, want empty", got.AITurn.SourceEventType)
	}
	if got.AITurn.SourceSceneID != "" {
		t.Fatalf("source scene id = %q, want empty", got.AITurn.SourceSceneID)
	}
	if got.AITurn.SourcePhaseID != "" {
		t.Fatalf("source phase id = %q, want empty", got.AITurn.SourcePhaseID)
	}
	if got.AITurn.LastError != "" {
		t.Fatalf("last error = %q, want empty", got.AITurn.LastError)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}
