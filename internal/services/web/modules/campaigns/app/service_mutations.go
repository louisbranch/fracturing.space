package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// createCampaign executes package-scoped creation behavior for this flow.
func (s service) createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
	}
	created, err := s.mutationGateway.CreateCampaign(ctx, input)
	if err != nil {
		return CreateCampaignResult{}, err
	}
	if strings.TrimSpace(created.CampaignID) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_campaign_id_was_empty", "created campaign id was empty")
	}
	return created, nil
}

// updateCampaign applies this package workflow transition.
func (s service) updateCampaign(ctx context.Context, campaignID string, input UpdateCampaignInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	if err := s.requireManageCampaign(ctx, campaignID); err != nil {
		return err
	}

	current, err := s.campaignWorkspace(ctx, campaignID)
	if err != nil {
		return err
	}

	patch := UpdateCampaignInput{}
	changed := false

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
		}
		if name != strings.TrimSpace(current.Name) {
			patch.Name = &name
			changed = true
		}
	}

	if input.ThemePrompt != nil {
		themePrompt := strings.TrimSpace(*input.ThemePrompt)
		if themePrompt != strings.TrimSpace(current.Theme) {
			patch.ThemePrompt = &themePrompt
			changed = true
		}
	}

	if input.Locale != nil {
		locale := campaignLocaleCanonical(*input.Locale)
		if locale == "" {
			return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_locale_value_is_invalid", "campaign locale value is invalid")
		}
		if locale != campaignLocaleCanonical(current.Locale) {
			patch.Locale = &locale
			changed = true
		}
	}

	if !changed {
		return nil
	}
	return s.mutationGateway.UpdateCampaign(ctx, campaignID, patch)
}

// startSession applies this package workflow transition.
func (s service) startSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	return s.mutationGateway.StartSession(ctx, campaignID, StartSessionInput{Name: strings.TrimSpace(input.Name)})
}

// endSession applies this package workflow transition.
func (s service) endSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.session_id_is_required", "session id is required")
	}
	return s.mutationGateway.EndSession(ctx, campaignID, EndSessionInput{SessionID: sessionID})
}

// createInvite executes package-scoped creation behavior for this flow.
func (s service) createInvite(ctx context.Context, campaignID string, input CreateInviteInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	return s.mutationGateway.CreateInvite(ctx, campaignID, CreateInviteInput{
		ParticipantID:   participantID,
		RecipientUserID: strings.TrimSpace(input.RecipientUserID),
	})
}

// revokeInvite applies this package workflow transition.
func (s service) revokeInvite(ctx context.Context, campaignID string, input RevokeInviteInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	inviteID := strings.TrimSpace(input.InviteID)
	if inviteID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.invite_id_is_required", "invite id is required")
	}
	return s.mutationGateway.RevokeInvite(ctx, campaignID, RevokeInviteInput{InviteID: inviteID})
}

// createCharacter executes package-scoped creation behavior for this flow.
func (s service) createCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	if input.Kind == CharacterKindUnspecified {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}
	if err := s.requirePolicy(ctx, campaignID, policyMutateCharacter); err != nil {
		return CreateCharacterResult{}, err
	}
	created, err := s.mutationGateway.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		return CreateCharacterResult{}, err
	}
	if strings.TrimSpace(created.CharacterID) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_character_id_was_empty", "created character id was empty")
	}
	return created, nil
}

// updateParticipant applies this package workflow transition.
func (s service) updateParticipant(ctx context.Context, campaignID string, input UpdateParticipantInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	role, ok := participantRoleCanonical(input.Role)
	if !ok {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_role_value_is_invalid", "participant role value is invalid")
	}
	name := strings.TrimSpace(input.Name)
	pronouns := strings.TrimSpace(input.Pronouns)
	requestedAccess := participantAccessCanonical(input.CampaignAccess)
	if strings.TrimSpace(input.CampaignAccess) != "" && requestedAccess == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_access_value_is_invalid", "campaign access value is invalid")
	}

	target := &campaignAuthorizationTarget{
		ResourceID:           participantID,
		TargetParticipantID:  participantID,
		ParticipantOperation: ParticipantGovernanceOperationMutate,
	}
	if requestedAccess != "" {
		target.RequestedCampaignAccess = requestedAccess
		target.ParticipantOperation = ParticipantGovernanceOperationAccessChange
	}
	if err := s.requireCampaignActionAccess(
		ctx,
		campaignID,
		campaignAuthzActionManage,
		campaignAuthzResourceParticipant,
		target,
		policyManageParticipant.denyKey,
		policyManageParticipant.denyMsg,
	); err != nil {
		return err
	}

	current, err := s.campaignParticipant(ctx, campaignID, participantID)
	if err != nil {
		return err
	}
	currentRole, _ := participantRoleCanonical(current.Role)
	currentAccess := participantAccessCanonical(current.CampaignAccess)
	if requestedAccess == currentAccess {
		requestedAccess = ""
	}
	if name == strings.TrimSpace(current.Name) &&
		role == currentRole &&
		pronouns == strings.TrimSpace(current.Pronouns) &&
		requestedAccess == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.at_least_one_participant_field_is_required", "at least one participant field is required")
	}

	return s.mutationGateway.UpdateParticipant(ctx, campaignID, UpdateParticipantInput{
		ParticipantID:  participantID,
		Name:           name,
		Role:           role,
		Pronouns:       pronouns,
		CampaignAccess: requestedAccess,
	})
}
