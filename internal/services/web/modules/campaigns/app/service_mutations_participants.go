package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CreateParticipant executes package-scoped creation behavior for this flow.
func (s participantMutationService) CreateParticipant(ctx context.Context, campaignID string, input CreateParticipantInput) (CreateParticipantResult, error) {
	return s.createParticipant(ctx, campaignID, input)
}

// UpdateParticipant applies this package workflow transition.
func (s participantMutationService) UpdateParticipant(ctx context.Context, campaignID string, input UpdateParticipantInput) error {
	return s.updateParticipant(ctx, campaignID, input)
}

// participantCreateRequest carries normalized participant creation values.
type participantCreateRequest struct {
	CampaignID     string
	Name           string
	Role           string
	CampaignAccess string
	Controller     string
}

// participantUpdateRequest carries normalized participant mutation values.
type participantUpdateRequest struct {
	CampaignID      string
	ParticipantID   string
	Name            string
	Role            string
	Pronouns        string
	RequestedAccess string
}

// createParticipant executes package-scoped creation behavior for this flow.
func (s participantMutationService) createParticipant(ctx context.Context, campaignID string, input CreateParticipantInput) (CreateParticipantResult, error) {
	request, err := normalizeParticipantCreateRequest(campaignID, input)
	if err != nil {
		return CreateParticipantResult{}, err
	}

	if err := s.auth.requireCampaignActionAccess(
		ctx,
		request.CampaignID,
		campaignAuthzActionManage,
		campaignAuthzResourceParticipant,
		participantCreateAuthorizationTarget(request),
		policyManageParticipant.denyKey,
		policyManageParticipant.denyMsg,
	); err != nil {
		return CreateParticipantResult{}, err
	}

	workspace, err := participantWorkspace(ctx, s.workspace, request.CampaignID)
	if err != nil {
		return CreateParticipantResult{}, err
	}
	if err := enforceParticipantCampaignGMModeInvariant(workspace.GMMode, request.Role, request.Controller); err != nil {
		return CreateParticipantResult{}, err
	}

	created, err := s.mutation.CreateParticipant(ctx, request.CampaignID, CreateParticipantInput{
		Name:           request.Name,
		Role:           request.Role,
		CampaignAccess: request.CampaignAccess,
	})
	if err != nil {
		return CreateParticipantResult{}, err
	}
	if strings.TrimSpace(created.ParticipantID) == "" {
		return CreateParticipantResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_participant_id_was_empty", "created participant id was empty")
	}
	return created, nil
}

// updateParticipant applies this package workflow transition.
func (s participantMutationService) updateParticipant(ctx context.Context, campaignID string, input UpdateParticipantInput) error {
	request, err := normalizeParticipantUpdateRequest(campaignID, input)
	if err != nil {
		return err
	}

	current, err := campaignParticipant(ctx, s.read, request.CampaignID, request.ParticipantID)
	if err != nil {
		return err
	}
	if !participantIsSelfOwned(ctx, current) {
		if err := s.auth.requireCampaignActionAccess(
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
	}
	request.RequestedAccess = normalizeRequestedParticipantAccess(request.RequestedAccess, current)
	workspace, err := participantWorkspace(ctx, s.workspace, request.CampaignID)
	if err != nil {
		return err
	}
	if err := enforceParticipantCampaignGMModeInvariant(workspace.GMMode, effectiveParticipantRole(request, current), current.Controller); err != nil {
		return err
	}
	if err := enforceAIControlledParticipantInvariant(request, current); err != nil {
		return err
	}
	if !participantUpdateHasChanges(request, current) {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.at_least_one_participant_field_is_required", "at least one participant field is required")
	}

	return s.mutation.UpdateParticipant(ctx, request.CampaignID, UpdateParticipantInput{
		ParticipantID:  request.ParticipantID,
		Name:           request.Name,
		Role:           request.Role,
		Pronouns:       request.Pronouns,
		CampaignAccess: request.RequestedAccess,
	})
}

// normalizeParticipantCreateRequest validates and normalizes participant-create input.
func normalizeParticipantCreateRequest(campaignID string, input CreateParticipantInput) (participantCreateRequest, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return participantCreateRequest{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return participantCreateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_name_is_required", "participant name is required")
	}

	role, ok := participantRoleCanonical(input.Role)
	if !ok {
		return participantCreateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_role_value_is_invalid", "participant role value is invalid")
	}

	campaignAccess := participantAccessCanonical(input.CampaignAccess)
	if campaignAccess == "" {
		return participantCreateRequest{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_access_value_is_invalid", "campaign access value is invalid")
	}

	return participantCreateRequest{
		CampaignID:     campaignID,
		Name:           name,
		Role:           role,
		CampaignAccess: campaignAccess,
		Controller:     participantControllerHuman,
	}, nil
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

// participantCreateAuthorizationTarget builds the authorization target for participant creation.
func participantCreateAuthorizationTarget(request participantCreateRequest) *AuthorizationTarget {
	if request.CampaignAccess == participantAccessMember {
		return nil
	}
	return &AuthorizationTarget{
		RequestedCampaignAccess: request.CampaignAccess,
		ParticipantOperation:    ParticipantGovernanceOperationAccessChange,
	}
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

// effectiveParticipantRole returns the role that would remain after the update request.
func effectiveParticipantRole(request participantUpdateRequest, current CampaignParticipant) string {
	if request.Role != "" {
		return request.Role
	}
	role, _ := participantRoleCanonical(current.Role)
	return role
}

// participantUpdateHasChanges reports whether a participant mutation changes at least one field.
func participantUpdateHasChanges(request participantUpdateRequest, current CampaignParticipant) bool {
	currentRole, _ := participantRoleCanonical(current.Role)
	return request.Name != strings.TrimSpace(current.Name) ||
		request.Role != currentRole ||
		request.Pronouns != strings.TrimSpace(current.Pronouns) ||
		request.RequestedAccess != ""
}

// enforceAIControlledParticipantInvariant rejects requests that would violate the fixed
// AI participant seat role/access contract.
func enforceAIControlledParticipantInvariant(request participantUpdateRequest, current CampaignParticipant) error {
	if participantControllerCanonical(current.Controller) != participantControllerAI {
		return nil
	}

	effectiveRole := effectiveParticipantRole(request, current)
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

// enforceParticipantCampaignGMModeInvariant rejects HUMAN GM seats when the
// campaign gm mode does not allow them.
func enforceParticipantCampaignGMModeInvariant(gmMode string, role string, controller string) error {
	if !campaignDisallowsHumanGMParticipants(gmMode) {
		return nil
	}
	if participantControllerCanonical(controller) != participantControllerHuman {
		return nil
	}
	if role != participantRoleGMValue {
		return nil
	}
	return apperrors.EK(
		apperrors.KindInvalidInput,
		"error.web.message.ai_gm_campaign_disallows_human_gm_participants",
		"AI GM campaigns cannot create or assign HUMAN GM participants",
	)
}
