package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// participantUpdateRequest carries normalized participant mutation values.
type participantUpdateRequest struct {
	CampaignID      string
	ParticipantID   string
	Name            string
	Role            string
	Pronouns        string
	RequestedAccess string
}

// updateParticipant applies this package workflow transition.
func (s service) updateParticipant(ctx context.Context, campaignID string, input UpdateParticipantInput) error {
	request, err := normalizeParticipantUpdateRequest(campaignID, input)
	if err != nil {
		return err
	}

	if err := s.requireCampaignActionAccess(
		ctx,
		request.CampaignID,
		campaignAuthzActionManage,
		campaignAuthzResourceParticipant,
		participantAuthorizationTarget(request),
		policyManageParticipant.denyKey,
		policyManageParticipant.denyMsg,
	); err != nil {
		return err
	}

	current, err := s.campaignParticipant(ctx, request.CampaignID, request.ParticipantID)
	if err != nil {
		return err
	}
	request.RequestedAccess = normalizeRequestedParticipantAccess(request.RequestedAccess, current)
	if err := enforceAIParticipantInvariant(request, current); err != nil {
		return err
	}
	if !participantUpdateHasChanges(request, current) {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.at_least_one_participant_field_is_required", "at least one participant field is required")
	}

	return s.mutationGateway.UpdateParticipant(ctx, request.CampaignID, UpdateParticipantInput{
		ParticipantID:  request.ParticipantID,
		Name:           request.Name,
		Role:           request.Role,
		Pronouns:       request.Pronouns,
		CampaignAccess: request.RequestedAccess,
	})
}

// normalizeParticipantUpdateRequest validates and normalizes participant mutation input.
func normalizeParticipantUpdateRequest(campaignID string, input UpdateParticipantInput) (participantUpdateRequest, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return participantUpdateRequest{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return participantUpdateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}

	role, ok := participantRoleCanonical(input.Role)
	if !ok {
		return participantUpdateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_role_value_is_invalid", "participant role value is invalid")
	}

	requestedAccess := participantAccessCanonical(input.CampaignAccess)
	if strings.TrimSpace(input.CampaignAccess) != "" && requestedAccess == "" {
		return participantUpdateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_access_value_is_invalid", "campaign access value is invalid")
	}

	return participantUpdateRequest{
		CampaignID:      campaignID,
		ParticipantID:   participantID,
		Name:            strings.TrimSpace(input.Name),
		Role:            role,
		Pronouns:        strings.TrimSpace(input.Pronouns),
		RequestedAccess: requestedAccess,
	}, nil
}

// participantAuthorizationTarget builds the authorization target for participant mutation.
func participantAuthorizationTarget(request participantUpdateRequest) *AuthorizationTarget {
	target := &AuthorizationTarget{
		ResourceID:           request.ParticipantID,
		TargetParticipantID:  request.ParticipantID,
		ParticipantOperation: ParticipantGovernanceOperationMutate,
	}
	if request.RequestedAccess != "" {
		target.RequestedCampaignAccess = request.RequestedAccess
		target.ParticipantOperation = ParticipantGovernanceOperationAccessChange
	}
	return target
}

// normalizeRequestedParticipantAccess clears no-op access changes against current state.
func normalizeRequestedParticipantAccess(requestedAccess string, current CampaignParticipant) string {
	currentAccess := participantAccessCanonical(current.CampaignAccess)
	if requestedAccess == currentAccess {
		return ""
	}
	return requestedAccess
}

// participantUpdateHasChanges reports whether a participant mutation changes at least one field.
func participantUpdateHasChanges(request participantUpdateRequest, current CampaignParticipant) bool {
	currentRole, _ := participantRoleCanonical(current.Role)
	return request.Name != strings.TrimSpace(current.Name) ||
		request.Role != currentRole ||
		request.Pronouns != strings.TrimSpace(current.Pronouns) ||
		request.RequestedAccess != ""
}

// enforceAIParticipantInvariant rejects requests that would violate the fixed
// AI participant seat role/access contract.
func enforceAIParticipantInvariant(request participantUpdateRequest, current CampaignParticipant) error {
	if participantControllerCanonical(current.Controller) != participantControllerAI {
		return nil
	}

	effectiveRole := request.Role
	if effectiveRole == "" {
		effectiveRole, _ = participantRoleCanonical(current.Role)
	}
	if effectiveRole == "" {
		effectiveRole = participantRoleGMValue
	}

	effectiveAccess := request.RequestedAccess
	if effectiveAccess == "" {
		effectiveAccess = participantAccessCanonical(current.CampaignAccess)
	}
	if effectiveAccess == "" {
		effectiveAccess = participantAccessMember
	}

	if effectiveRole == participantRoleGMValue && effectiveAccess == participantAccessMember {
		return nil
	}
	return apperrors.EK(
		apperrors.KindConflict,
		"error.web.message.participant_ai_role_and_access_are_fixed",
		"AI participants must remain GM and Member",
	)
}
