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
