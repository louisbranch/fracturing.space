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

func TestApplySessionOOCOpened(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	// Seed with pre-existing state to verify fields are overwritten.
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		OOCPosts: []storage.SessionOOCPost{
			{PostID: "old-post", ParticipantID: "p1", Body: "stale"},
		},
		ReadyToResumeParticipantIDs: []string{"p1"},
		OOCResolutionPending:        true,
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 0, 0, 0, time.UTC)

	err := applier.applySessionOOCOpened(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCOpenedPayload{
		RequestedByParticipantID: ids.ParticipantID("p2"),
		Reason:                   "need a break",
		InterruptedSceneID:       ids.SceneID("scene-1"),
		InterruptedPhaseID:       "phase-1",
		InterruptedPhaseStatus:   "players",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if !got.OOCPaused {
		t.Fatal("ooc paused = false, want true")
	}
	if got.OOCRequestedByParticipantID != "p2" {
		t.Fatalf("requested by = %q, want %q", got.OOCRequestedByParticipantID, "p2")
	}
	if got.OOCReason != "need a break" {
		t.Fatalf("reason = %q, want %q", got.OOCReason, "need a break")
	}
	if got.OOCInterruptedSceneID != "scene-1" {
		t.Fatalf("interrupted scene id = %q, want %q", got.OOCInterruptedSceneID, "scene-1")
	}
	if got.OOCInterruptedPhaseID != "phase-1" {
		t.Fatalf("interrupted phase id = %q, want %q", got.OOCInterruptedPhaseID, "phase-1")
	}
	if got.OOCInterruptedPhaseStatus != "players" {
		t.Fatalf("interrupted phase status = %q, want %q", got.OOCInterruptedPhaseStatus, "players")
	}
	if got.OOCResolutionPending {
		t.Fatal("resolution pending = true, want false after open")
	}
	if len(got.OOCPosts) != 0 {
		t.Fatalf("ooc posts = %#v, want empty", got.OOCPosts)
	}
	if len(got.ReadyToResumeParticipantIDs) != 0 {
		t.Fatalf("ready to resume = %#v, want empty", got.ReadyToResumeParticipantIDs)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionOOCPosted(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		OOCPaused:  true,
		OOCPosts:   []storage.SessionOOCPost{},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 1, 0, 0, time.UTC)

	err := applier.applySessionOOCPosted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCPostedPayload{
		PostID:        "post-1",
		ParticipantID: ids.ParticipantID("p1"),
		Body:          "Let's discuss strategy.",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if len(got.OOCPosts) != 1 {
		t.Fatalf("ooc posts count = %d, want 1", len(got.OOCPosts))
	}
	post := got.OOCPosts[0]
	if post.PostID != "post-1" {
		t.Fatalf("post id = %q, want %q", post.PostID, "post-1")
	}
	if post.ParticipantID != "p1" {
		t.Fatalf("participant id = %q, want %q", post.ParticipantID, "p1")
	}
	if post.Body != "Let's discuss strategy." {
		t.Fatalf("body = %q, want %q", post.Body, "Let's discuss strategy.")
	}
	if !post.CreatedAt.Equal(now) {
		t.Fatalf("created at = %v, want %v", post.CreatedAt, now)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionOOCReadyMarked(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID:                  "camp-1",
		SessionID:                   "sess-1",
		OOCPaused:                   true,
		ReadyToResumeParticipantIDs: []string{},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 2, 0, 0, time.UTC)

	err := applier.applySessionOOCReadyMarked(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCReadyMarkedPayload{
		ParticipantID: ids.ParticipantID("p1"),
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if len(got.ReadyToResumeParticipantIDs) != 1 || got.ReadyToResumeParticipantIDs[0] != "p1" {
		t.Fatalf("ready to resume = %#v, want [p1]", got.ReadyToResumeParticipantIDs)
	}

	// Marking the same participant again should be idempotent.
	err = applier.applySessionOOCReadyMarked(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now.Add(time.Second),
	}, session.OOCReadyMarkedPayload{
		ParticipantID: ids.ParticipantID("p1"),
	})
	if err != nil {
		t.Fatalf("apply duplicate: %v", err)
	}

	got = store.interactions["camp-1:sess-1"]
	if len(got.ReadyToResumeParticipantIDs) != 1 {
		t.Fatalf("ready to resume after duplicate = %#v, want [p1]", got.ReadyToResumeParticipantIDs)
	}
}

func TestApplySessionOOCReadyCleared(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID:                  "camp-1",
		SessionID:                   "sess-1",
		OOCPaused:                   true,
		ReadyToResumeParticipantIDs: []string{"p1", "p2"},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 3, 0, 0, time.UTC)

	err := applier.applySessionOOCReadyCleared(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCReadyClearedPayload{
		ParticipantID: ids.ParticipantID("p1"),
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if len(got.ReadyToResumeParticipantIDs) != 1 || got.ReadyToResumeParticipantIDs[0] != "p2" {
		t.Fatalf("ready to resume = %#v, want [p2]", got.ReadyToResumeParticipantIDs)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionOOCClosed(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID:                  "camp-1",
		SessionID:                   "sess-1",
		OOCPaused:                   true,
		OOCInterruptedSceneID:       "scene-1",
		OOCInterruptedPhaseID:       "phase-1",
		OOCInterruptedPhaseStatus:   "players",
		OOCPosts:                    []storage.SessionOOCPost{{PostID: "post-1", Body: "hi"}},
		ReadyToResumeParticipantIDs: []string{"p1"},
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 4, 0, 0, time.UTC)

	err := applier.applySessionOOCClosed(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCClosedPayload{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.OOCPaused {
		t.Fatal("ooc paused = true, want false after close")
	}
	if len(got.OOCPosts) != 0 {
		t.Fatalf("ooc posts = %#v, want empty", got.OOCPosts)
	}
	if len(got.ReadyToResumeParticipantIDs) != 0 {
		t.Fatalf("ready to resume = %#v, want empty", got.ReadyToResumeParticipantIDs)
	}
	// When interrupted scene and phase were set, resolution should be pending.
	if !got.OOCResolutionPending {
		t.Fatal("resolution pending = false, want true when interrupted scene+phase were set")
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestApplySessionOOCClosedNoResolutionWhenNoInterruption(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		OOCPaused:  true,
		// No interrupted scene/phase set.
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 4, 30, 0, time.UTC)

	err := applier.applySessionOOCClosed(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCClosedPayload{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.OOCResolutionPending {
		// Invariant: resolution is only pending when both interrupted scene and phase were set.
		t.Fatal("resolution pending = true, want false when no interrupted scene+phase")
	}
}

func TestApplySessionOOCResolved(t *testing.T) {
	t.Parallel()

	store := newFakeSessionInteractionStore()
	store.interactions["camp-1:sess-1"] = storage.SessionInteraction{
		CampaignID:                  "camp-1",
		SessionID:                   "sess-1",
		OOCRequestedByParticipantID: "p2",
		OOCReason:                   "need a break",
		OOCInterruptedSceneID:       "scene-1",
		OOCInterruptedPhaseID:       "phase-1",
		OOCInterruptedPhaseStatus:   "players",
		OOCResolutionPending:        true,
	}
	applier := Applier{SessionInteraction: store}
	now := time.Date(2026, 3, 27, 13, 5, 0, 0, time.UTC)

	err := applier.applySessionOOCResolved(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		Timestamp:  now,
	}, session.OOCResolvedPayload{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:sess-1"]
	if got.OOCRequestedByParticipantID != "" {
		t.Fatalf("requested by = %q, want empty", got.OOCRequestedByParticipantID)
	}
	if got.OOCReason != "" {
		t.Fatalf("reason = %q, want empty", got.OOCReason)
	}
	if got.OOCInterruptedSceneID != "" {
		t.Fatalf("interrupted scene id = %q, want empty", got.OOCInterruptedSceneID)
	}
	if got.OOCInterruptedPhaseID != "" {
		t.Fatalf("interrupted phase id = %q, want empty", got.OOCInterruptedPhaseID)
	}
	if got.OOCInterruptedPhaseStatus != "" {
		t.Fatalf("interrupted phase status = %q, want empty", got.OOCInterruptedPhaseStatus)
	}
	if got.OOCResolutionPending {
		t.Fatal("resolution pending = true, want false after resolve")
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", got.UpdatedAt, now)
	}
}
