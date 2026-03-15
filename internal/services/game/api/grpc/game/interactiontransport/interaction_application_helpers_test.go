package interactiontransport

import (
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSessionInteractionToProtoSortsPostsAndReadyParticipants(t *testing.T) {
	t.Parallel()

	earlier := time.Unix(100, 0).UTC()
	later := time.Unix(200, 0).UTC()
	got := sessionInteractionToProto(storage.SessionInteraction{
		OOCPaused: true,
		OOCPosts: []storage.SessionOOCPost{
			{PostID: "later", ParticipantID: "p2", Body: "second", CreatedAt: later},
			{PostID: "earlier", ParticipantID: "p1", Body: "first", CreatedAt: earlier},
		},
		ReadyToResumeParticipantIDs: []string{"p2", "p1"},
	})

	if !got.GetOpen() {
		t.Fatal("expected ooc state to be open")
	}
	if len(got.GetPosts()) != 2 || got.GetPosts()[0].GetPostId() != "earlier" || got.GetPosts()[1].GetPostId() != "later" {
		t.Fatalf("posts = %#v, want chronological order", got.GetPosts())
	}
	if len(got.GetReadyToResumeParticipantIds()) != 2 || got.GetReadyToResumeParticipantIds()[0] != "p1" || got.GetReadyToResumeParticipantIds()[1] != "p2" {
		t.Fatalf("ready ids = %#v, want sorted order", got.GetReadyToResumeParticipantIds())
	}
}

func TestSceneInteractionToProtoReturnsGMStateWhenClosed(t *testing.T) {
	t.Parallel()

	got := sceneInteractionToProto(storage.SceneInteraction{})

	if got.GetPhaseId() != "" {
		t.Fatalf("phase id = %q, want empty", got.GetPhaseId())
	}
	if len(got.GetActingCharacterIds()) != 0 || len(got.GetActingParticipantIds()) != 0 || len(got.GetSlots()) != 0 {
		t.Fatalf("closed phase should expose empty collections: %#v", got)
	}
}

func TestSceneInteractionToProtoSortsCollectionsForPlayersPhase(t *testing.T) {
	t.Parallel()

	earlier := time.Unix(100, 0).UTC()
	later := time.Unix(200, 0).UTC()
	got := sceneInteractionToProto(storage.SceneInteraction{
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusPlayers,
		FrameText:            "What do you do?",
		ActingCharacterIDs:   []string{"char-b", "char-a"},
		ActingParticipantIDs: []string{"p2", "p1"},
		Slots: []storage.ScenePlayerSlot{
			{ParticipantID: "p2", SummaryText: "later participant", CharacterIDs: []string{"char-b"}, UpdatedAt: later, Yielded: true},
			{ParticipantID: "p1", SummaryText: "earlier participant", CharacterIDs: []string{"char-a"}, UpdatedAt: earlier, Yielded: false},
		},
	})

	if got.GetPhaseId() != "phase-1" || got.GetFrameText() != "What do you do?" {
		t.Fatalf("phase metadata = %#v", got)
	}
	if got.GetStatus() != gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS {
		t.Fatalf("status = %v, want PLAYERS", got.GetStatus())
	}
	if got.GetActingCharacterIds()[0] != "char-a" || got.GetActingParticipantIds()[0] != "p1" {
		t.Fatalf("acting sets not sorted: %#v %#v", got.GetActingCharacterIds(), got.GetActingParticipantIds())
	}
	if got.GetSlots()[0].GetParticipantId() != "p1" || got.GetSlots()[0].GetSummaryText() != "earlier participant" {
		t.Fatalf("slots = %#v, want sorted by participant then time", got.GetSlots())
	}
	if !got.GetSlots()[1].GetYielded() {
		t.Fatalf("slot yielded = %#v, want second slot yielded", got.GetSlots())
	}
}

func TestSceneInteractionToProtoMapsReviewStatus(t *testing.T) {
	t.Parallel()

	got := sceneInteractionToProto(storage.SceneInteraction{
		PhaseOpen:   true,
		PhaseID:     "phase-2",
		PhaseStatus: scene.PlayerPhaseStatusGMReview,
		Slots: []storage.ScenePlayerSlot{
			{
				ParticipantID:      "p1",
				ReviewStatus:       scene.PlayerPhaseSlotReviewStatusChangesRequested,
				ReviewReason:       "Corin does not know Fireball",
				ReviewCharacterIDs: []string{"char-1"},
			},
		},
	})

	if got.GetStatus() != gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW {
		t.Fatalf("status = %v, want GM_REVIEW", got.GetStatus())
	}
	if got.GetSlots()[0].GetReviewStatus() != gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED {
		t.Fatalf("review status = %v", got.GetSlots()[0].GetReviewStatus())
	}
	if got.GetSlots()[0].GetReviewReason() != "Corin does not know Fireball" {
		t.Fatalf("review reason = %q", got.GetSlots()[0].GetReviewReason())
	}
}
