package interactiontransport

import (
	"slices"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func deriveInteractionControlState(
	actor storage.ParticipantRecord,
	sessionInteraction storage.SessionInteraction,
	sceneInteraction storage.SceneInteraction,
) *campaignv1.InteractionControlState {
	control := &campaignv1.InteractionControlState{
		Mode:                  campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_UNSPECIFIED,
		AllowedTransitions:    []campaignv1.InteractionTransition{},
		RecommendedTransition: campaignv1.InteractionTransition_INTERACTION_TRANSITION_UNSPECIFIED,
	}
	if strings.TrimSpace(sessionInteraction.SessionID) == "" {
		return control
	}

	addAllowed := newInteractionTransitionSet(control)
	isAuthority := strings.TrimSpace(actor.ID) != "" && strings.TrimSpace(actor.ID) == strings.TrimSpace(sessionInteraction.GMAuthorityParticipantID)
	isActingParticipant := slices.Contains(sceneInteraction.ActingParticipantIDs, actor.ID)
	isYielded := false
	hasSubmittedAction := false
	for _, slot := range sceneInteraction.Slots {
		if strings.TrimSpace(slot.ParticipantID) != strings.TrimSpace(actor.ID) {
			continue
		}
		isYielded = slot.Yielded
		hasSubmittedAction = strings.TrimSpace(slot.SummaryText) != "" &&
			slot.ReviewStatus != "changes_requested"
		break
	}

	if sessionInteraction.OOCPaused {
		control.Mode = campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_OOC
		addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_POST_SESSION_OOC)
		if slices.Contains(sessionInteraction.ReadyToResumeParticipantIDs, actor.ID) {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_CLEAR_OOC_READY_TO_RESUME)
			control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_POST_SESSION_OOC
		} else {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_MARK_OOC_READY_TO_RESUME)
			control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_MARK_OOC_READY_TO_RESUME
		}
		if isAuthority {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_RESOLVE_SESSION_OOC)
		}
		control.RecommendedSceneId = strings.TrimSpace(sessionInteraction.OOCInterruptedSceneID)
		control.RecommendedPhaseId = strings.TrimSpace(sessionInteraction.OOCInterruptedPhaseID)
		return control
	}
	if sessionInteraction.OOCResolutionPending {
		control.Mode = campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_OOC_RESOLUTION
		control.RecommendedSceneId = strings.TrimSpace(sessionInteraction.OOCInterruptedSceneID)
		control.RecommendedPhaseId = strings.TrimSpace(sessionInteraction.OOCInterruptedPhaseID)
		if isAuthority {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_RESOLVE_SESSION_OOC)
			control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_RESOLVE_SESSION_OOC
		}
		return control
	}

	switch sceneInteraction.PhaseStatus {
	case "players":
		control.Mode = campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_PLAYERS
		control.RecommendedSceneId = strings.TrimSpace(sceneInteraction.SceneID)
		control.RecommendedPhaseId = strings.TrimSpace(sceneInteraction.PhaseID)
		addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SESSION_OOC)
		if isActingParticipant {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_SUBMIT_SCENE_PLAYER_ACTION)
			if isYielded {
				addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_WITHDRAW_SCENE_PLAYER_YIELD)
			} else {
				addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_YIELD_SCENE_PLAYER_PHASE)
			}
			if hasSubmittedAction {
				control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_YIELD_SCENE_PLAYER_PHASE
				if isYielded {
					control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_WITHDRAW_SCENE_PLAYER_YIELD
				}
			} else {
				control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_SUBMIT_SCENE_PLAYER_ACTION
			}
		}
		if isAuthority {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_INTERRUPT_SCENE_PLAYER_PHASE)
		}
	case "gm_review":
		control.Mode = campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_GM_REVIEW
		control.RecommendedSceneId = strings.TrimSpace(sceneInteraction.SceneID)
		control.RecommendedPhaseId = strings.TrimSpace(sceneInteraction.PhaseID)
		addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SESSION_OOC)
		if isAuthority {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_RESOLVE_SCENE_PLAYER_REVIEW)
			control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_RESOLVE_SCENE_PLAYER_REVIEW
		}
	default:
		control.Mode = campaignv1.InteractionControlMode_INTERACTION_CONTROL_MODE_GM
		addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SESSION_OOC)
		if isAuthority {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_ACTIVATE_SCENE)
			if strings.TrimSpace(sessionInteraction.ActiveSceneID) != "" {
				control.RecommendedSceneId = strings.TrimSpace(sessionInteraction.ActiveSceneID)
				addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_RECORD_SCENE_GM_INTERACTION)
				addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SCENE_PLAYER_PHASE)
				control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_OPEN_SCENE_PLAYER_PHASE
			} else {
				control.RecommendedTransition = campaignv1.InteractionTransition_INTERACTION_TRANSITION_ACTIVATE_SCENE
			}
			if sessionInteraction.AITurn.Status == "failed" {
				addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_RETRY_AI_GM_TURN)
			}
		}
		if actor.CampaignAccess == participant.CampaignAccessOwner || actor.CampaignAccess == participant.CampaignAccessManager {
			addAllowed(campaignv1.InteractionTransition_INTERACTION_TRANSITION_SET_SESSION_GM_AUTHORITY)
		}
	}

	return control
}

func newInteractionTransitionSet(control *campaignv1.InteractionControlState) func(campaignv1.InteractionTransition) {
	seen := map[campaignv1.InteractionTransition]struct{}{}
	return func(value campaignv1.InteractionTransition) {
		if control == nil || value == campaignv1.InteractionTransition_INTERACTION_TRANSITION_UNSPECIFIED {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		control.AllowedTransitions = append(control.AllowedTransitions, value)
	}
}
