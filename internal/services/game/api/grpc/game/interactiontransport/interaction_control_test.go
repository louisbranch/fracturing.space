package interactiontransport

import (
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDeriveInteractionControlStateAllowsAuthoritativeGMMemberTransitions(t *testing.T) {
	t.Parallel()

	control := deriveInteractionControlState(
		storage.ParticipantRecord{
			ID:             "gm-1",
			CampaignAccess: participant.CampaignAccessMember,
		},
		storage.SessionInteraction{
			SessionID:                "sess-1",
			ActiveSceneID:            "scene-1",
			GMAuthorityParticipantID: "gm-1",
		},
		storage.SceneInteraction{},
	)

	if control.GetMode() != gamev1.InteractionControlMode_INTERACTION_CONTROL_MODE_GM {
		t.Fatalf("mode = %v, want GM", control.GetMode())
	}
	if control.GetRecommendedTransition() != gamev1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SCENE_PLAYER_PHASE {
		t.Fatalf("recommended_transition = %v, want OPEN_SCENE_PLAYER_PHASE", control.GetRecommendedTransition())
	}
	if !containsInteractionTransition(control.GetAllowedTransitions(), gamev1.InteractionTransition_INTERACTION_TRANSITION_ACTIVATE_SCENE) {
		t.Fatalf("allowed_transitions = %v, want ACTIVATE_SCENE", control.GetAllowedTransitions())
	}
	if !containsInteractionTransition(control.GetAllowedTransitions(), gamev1.InteractionTransition_INTERACTION_TRANSITION_RECORD_SCENE_GM_INTERACTION) {
		t.Fatalf("allowed_transitions = %v, want RECORD_SCENE_GM_INTERACTION", control.GetAllowedTransitions())
	}
	if containsInteractionTransition(control.GetAllowedTransitions(), gamev1.InteractionTransition_INTERACTION_TRANSITION_SET_SESSION_GM_AUTHORITY) {
		t.Fatalf("allowed_transitions = %v, did not expect SET_SESSION_GM_AUTHORITY for member gm", control.GetAllowedTransitions())
	}
}

func TestDeriveInteractionControlStateRecommendsSubmitAfterChangesRequested(t *testing.T) {
	t.Parallel()

	control := deriveInteractionControlState(
		storage.ParticipantRecord{ID: "player-1"},
		storage.SessionInteraction{
			SessionID:                "sess-1",
			ActiveSceneID:            "scene-1",
			GMAuthorityParticipantID: "gm-1",
		},
		storage.SceneInteraction{
			SceneID:              "scene-1",
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusPlayers,
			ActingParticipantIDs: []string{"player-1"},
			Slots: []storage.ScenePlayerSlot{{
				ParticipantID: "player-1",
				SummaryText:   "Aria advances.",
				Yielded:       true,
				ReviewStatus:  scene.PlayerPhaseSlotReviewStatusChangesRequested,
			}},
		},
	)

	if control.GetMode() != gamev1.InteractionControlMode_INTERACTION_CONTROL_MODE_PLAYERS {
		t.Fatalf("mode = %v, want PLAYERS", control.GetMode())
	}
	if control.GetRecommendedTransition() != gamev1.InteractionTransition_INTERACTION_TRANSITION_SUBMIT_SCENE_PLAYER_ACTION {
		t.Fatalf("recommended_transition = %v, want SUBMIT_SCENE_PLAYER_ACTION", control.GetRecommendedTransition())
	}
}

func containsInteractionTransition(values []gamev1.InteractionTransition, want gamev1.InteractionTransition) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
