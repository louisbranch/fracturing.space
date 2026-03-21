package projection

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyScenePlayerPhasePostedClearsYieldedAndReviewState(t *testing.T) {
	t.Parallel()

	store := newFakeSceneInteractionStore()
	store.interactions["camp-1:scene-1"] = storage.SceneInteraction{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		PhaseOpen:  true,
		PhaseID:    "phase-1",
		Slots: []storage.ScenePlayerSlot{{
			ParticipantID:      "p1",
			SummaryText:        "Old",
			CharacterIDs:       []string{"char-1"},
			Yielded:            true,
			ReviewStatus:       scene.PlayerPhaseSlotReviewStatusChangesRequested,
			ReviewReason:       "Fix this.",
			ReviewCharacterIDs: []string{"char-1"},
		}},
	}
	applier := Applier{SceneInteraction: store}

	err := applier.applyScenePlayerPhasePosted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
	}, scene.PlayerPhasePostedPayload{
		SceneID:       ids.SceneID("scene-1"),
		PhaseID:       "phase-1",
		ParticipantID: ids.ParticipantID("p1"),
		CharacterIDs:  []ids.CharacterID{"char-1"},
		SummaryText:   "Corrected",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:scene-1"].Slots[0]
	if got.Yielded {
		t.Fatal("slot yielded = true, want false after repost")
	}
	if got.ReviewStatus != scene.PlayerPhaseSlotReviewStatusOpen {
		t.Fatalf("review status = %q, want %q", got.ReviewStatus, scene.PlayerPhaseSlotReviewStatusOpen)
	}
	if got.ReviewReason != "" {
		t.Fatalf("review reason = %q, want empty", got.ReviewReason)
	}
	if len(got.ReviewCharacterIDs) != 0 {
		t.Fatalf("review character ids = %#v, want empty", got.ReviewCharacterIDs)
	}
}

func TestApplyScenePlayerPhaseStartedInitializesPhaseStateAndSlots(t *testing.T) {
	t.Parallel()

	store := newFakeSceneInteractionStore()
	applier := Applier{SceneInteraction: store}

	err := applier.applyScenePlayerPhaseStarted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 5, 0, 0, time.UTC),
	}, scene.PlayerPhaseStartedPayload{
		SceneID:              ids.SceneID("scene-1"),
		PhaseID:              "phase-1",
		ActingCharacterIDs:   []ids.CharacterID{"char-1", "char-2"},
		ActingParticipantIDs: []ids.ParticipantID{"p2", "p1"},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:scene-1"]
	if !got.PhaseOpen || got.PhaseID != "phase-1" {
		t.Fatalf("phase state = %#v", got)
	}
	if got.SessionID != "sess-1" {
		t.Fatalf("session id = %q, want sess-1", got.SessionID)
	}
	if got.PhaseStatus != scene.PlayerPhaseStatusPlayers {
		t.Fatalf("phase status = %q, want %q", got.PhaseStatus, scene.PlayerPhaseStatusPlayers)
	}
	if len(got.Slots) != 2 {
		t.Fatalf("slots = %#v, want 2 initialized slots", got.Slots)
	}
}

func TestApplySceneGMInteractionCommittedStoresInteractionHistory(t *testing.T) {
	t.Parallel()

	store := newFakeSceneGMInteractionStore()
	applier := Applier{SceneGMInteraction: store}
	now := time.Date(2026, 3, 13, 10, 10, 0, 0, time.UTC)

	err := applier.applySceneGMInteractionCommitted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		SceneID:    "scene-1",
		Timestamp:  now,
	}, scene.GMInteractionCommittedPayload{
		SceneID:       ids.SceneID("scene-1"),
		InteractionID: "interaction-1",
		PhaseID:       "phase-1",
		ParticipantID: ids.ParticipantID("gm-ai"),
		Title:         "Chamber Quiets",
		CharacterIDs:  []ids.CharacterID{"char-1"},
		Beats: []scene.GMInteractionBeat{{
			BeatID: "beat-1",
			Type:   scene.GMInteractionBeatTypeFiction,
			Text:   "The chamber falls quiet.",
		}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := store.interactions["camp-1:scene-1"]
	if len(got) != 1 {
		t.Fatalf("gm interactions = %#v, want one record", got)
	}
	if got[0].Title != "Chamber Quiets" || got[0].ParticipantID != "gm-ai" {
		t.Fatalf("gm interaction = %#v", got[0])
	}
	if !got[0].CreatedAt.Equal(now) {
		t.Fatalf("gm interaction created at = %#v, want %v", got[0].CreatedAt, now)
	}
}

func TestApplyScenePlayerPhaseReviewLifecycle(t *testing.T) {
	t.Parallel()

	store := newFakeSceneInteractionStore()
	store.interactions["camp-1:scene-1"] = storage.SceneInteraction{
		CampaignID:           "camp-1",
		SceneID:              "scene-1",
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusPlayers,
		ActingCharacterIDs:   []string{"char-1", "char-2"},
		ActingParticipantIDs: []string{"p1", "p2"},
		Slots: []storage.ScenePlayerSlot{
			{ParticipantID: "p1", SummaryText: "Aria advances.", CharacterIDs: []string{"char-1"}, Yielded: true, ReviewStatus: scene.PlayerPhaseSlotReviewStatusOpen},
			{ParticipantID: "p2", SummaryText: "Borin covers the flank.", CharacterIDs: []string{"char-2"}, Yielded: true, ReviewStatus: scene.PlayerPhaseSlotReviewStatusOpen},
		},
	}
	applier := Applier{SceneInteraction: store}

	err := applier.applyScenePlayerPhaseReviewStarted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 6, 0, 0, time.UTC),
	}, scene.PlayerPhaseReviewStartedPayload{
		SceneID: ids.SceneID("scene-1"),
		PhaseID: "phase-1",
	})
	if err != nil {
		t.Fatalf("apply review started: %v", err)
	}

	got := store.interactions["camp-1:scene-1"]
	if got.PhaseStatus != scene.PlayerPhaseStatusGMReview {
		t.Fatalf("phase status = %q, want %q", got.PhaseStatus, scene.PlayerPhaseStatusGMReview)
	}
	for _, slot := range got.Slots {
		if slot.ReviewStatus != scene.PlayerPhaseSlotReviewStatusUnderReview {
			t.Fatalf("review started slot = %#v", slot)
		}
	}

	err = applier.applyScenePlayerPhaseRevisionsRequested(context.Background(), event.Event{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 7, 0, 0, time.UTC),
	}, scene.PlayerPhaseRevisionsRequestedPayload{
		SceneID: ids.SceneID("scene-1"),
		PhaseID: "phase-1",
		Revisions: []scene.PlayerPhaseRevisionRequest{{
			ParticipantID: ids.ParticipantID("p1"),
			Reason:        "Aria cannot cast that spell.",
			CharacterIDs:  []ids.CharacterID{"char-1"},
		}},
	})
	if err != nil {
		t.Fatalf("apply revisions requested: %v", err)
	}

	got = store.interactions["camp-1:scene-1"]
	if got.PhaseStatus != scene.PlayerPhaseStatusPlayers {
		t.Fatalf("phase status after revisions = %q, want %q", got.PhaseStatus, scene.PlayerPhaseStatusPlayers)
	}
	if got.Slots[0].ParticipantID == "p1" {
		if got.Slots[0].Yielded || got.Slots[0].ReviewStatus != scene.PlayerPhaseSlotReviewStatusChangesRequested {
			t.Fatalf("targeted slot = %#v", got.Slots[0])
		}
		if got.Slots[1].ReviewStatus != scene.PlayerPhaseSlotReviewStatusAccepted {
			t.Fatalf("untargeted slot = %#v", got.Slots[1])
		}
	} else {
		if got.Slots[1].Yielded || got.Slots[1].ReviewStatus != scene.PlayerPhaseSlotReviewStatusChangesRequested {
			t.Fatalf("targeted slot = %#v", got.Slots[1])
		}
		if got.Slots[0].ReviewStatus != scene.PlayerPhaseSlotReviewStatusAccepted {
			t.Fatalf("untargeted slot = %#v", got.Slots[0])
		}
	}

	err = applier.applyScenePlayerPhaseAccepted(context.Background(), event.Event{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 8, 0, 0, time.UTC),
	}, scene.PlayerPhaseAcceptedPayload{
		SceneID: ids.SceneID("scene-1"),
		PhaseID: "phase-1",
	})
	if err != nil {
		t.Fatalf("apply accepted: %v", err)
	}

	for _, slot := range store.interactions["camp-1:scene-1"].Slots {
		if slot.ReviewStatus != scene.PlayerPhaseSlotReviewStatusAccepted {
			t.Fatalf("accepted slot = %#v", slot)
		}
	}

	err = applier.applyScenePlayerPhaseEnded(context.Background(), event.Event{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		Timestamp:  time.Date(2026, 3, 13, 10, 9, 0, 0, time.UTC),
	}, scene.PlayerPhaseEndedPayload{
		SceneID: ids.SceneID("scene-1"),
		PhaseID: "phase-1",
		Reason:  "accepted",
	})
	if err != nil {
		t.Fatalf("apply ended: %v", err)
	}

	got = store.interactions["camp-1:scene-1"]
	if got.PhaseOpen || got.PhaseID != "" || len(got.Slots) != 0 {
		t.Fatalf("ended phase = %#v", got)
	}
}
