package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const (
	participantDeleteReasonTargetIsAI                 = "AUTHZ_DENY_TARGET_IS_AI_PARTICIPANT"
	participantDeleteReasonTargetOwnsActiveCharacters = "AUTHZ_DENY_TARGET_OWNS_ACTIVE_CHARACTERS"
	participantDeleteReasonTargetControlsCharacters   = "AUTHZ_DENY_TARGET_CONTROLS_ACTIVE_CHARACTERS"
)

// participantDeleteAuthorizationTarget scopes authz checks for participant removal.
func participantDeleteAuthorizationTarget(participant CampaignParticipant) *AuthorizationTarget {
	return &AuthorizationTarget{
		ResourceID:           strings.TrimSpace(participant.ID),
		TargetParticipantID:  strings.TrimSpace(participant.ID),
		TargetCampaignAccess: participantAccessCanonical(participant.CampaignAccess),
		ParticipantOperation: ParticipantGovernanceOperationRemove,
	}
}

// participantDeleteDecision centralizes the authz lookup used by both the edit
// page and the delete mutation flow.
func participantDeleteDecision(
	ctx context.Context,
	auth AuthorizationGateway,
	campaignID string,
	participant CampaignParticipant,
) (AuthorizationDecision, error) {
	if auth == nil {
		return AuthorizationDecision{}, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return AuthorizationDecision{}, nil
	}
	return auth.CanCampaignAction(
		ctx,
		campaignID,
		campaignAuthzActionManage,
		campaignAuthzResourceParticipant,
		participantDeleteAuthorizationTarget(participant),
	)
}

// participantDeleteStateFromDecision maps delete authz into render-owned danger-zone state.
func participantDeleteStateFromDecision(participant CampaignParticipant, decision AuthorizationDecision) CampaignParticipantDeleteState {
	state := CampaignParticipantDeleteState{
		HasAssociatedUser: strings.TrimSpace(participant.UserID) != "",
	}
	reason := strings.TrimSpace(decision.ReasonCode)
	switch reason {
	case participantDeleteReasonTargetIsAI:
		return state
	case participantDeleteReasonTargetOwnsActiveCharacters:
		state.Visible = true
		state.BlockedByOwnedCharacters = true
	case participantDeleteReasonTargetControlsCharacters:
		state.Visible = true
		state.BlockedByControlledCharacters = true
	default:
		if decision.Evaluated && decision.Allowed {
			state.Visible = true
			state.Enabled = true
			return state
		}
		return state
	}
	state.Enabled = false
	return state
}

// participantDeleteError maps participant-remove authz outcomes into
// user-facing web errors for direct POST submissions.
func participantDeleteError(decision AuthorizationDecision) error {
	switch strings.TrimSpace(decision.ReasonCode) {
	case participantDeleteReasonTargetIsAI:
		return apperrors.EK(apperrors.KindConflict, "error.web.message.ai_participants_cannot_be_deleted", "AI participants cannot be deleted")
	case participantDeleteReasonTargetOwnsActiveCharacters:
		return apperrors.EK(apperrors.KindConflict, "error.web.message.participant_owns_active_characters", "participant owns active characters")
	case participantDeleteReasonTargetControlsCharacters:
		return apperrors.EK(apperrors.KindConflict, "error.web.message.participant_controls_active_characters", "participant controls active characters")
	default:
		return apperrors.EK(apperrors.KindForbidden, policyManageParticipant.denyKey, policyManageParticipant.denyMsg)
	}
}
