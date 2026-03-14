package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CreateInvite executes package-scoped creation behavior for this flow.
func (s inviteMutationService) CreateInvite(ctx context.Context, campaignID string, input CreateInviteInput) error {
	return s.createInvite(ctx, campaignID, input)
}

// RevokeInvite applies this package workflow transition.
func (s inviteMutationService) RevokeInvite(ctx context.Context, campaignID string, input RevokeInviteInput) error {
	return s.revokeInvite(ctx, campaignID, input)
}

// createInvite executes package-scoped creation behavior for this flow.
func (s inviteMutationService) createInvite(ctx context.Context, campaignID string, input CreateInviteInput) error {
	if err := s.auth.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	return s.mutation.CreateInvite(ctx, campaignID, CreateInviteInput{
		ParticipantID:     participantID,
		RecipientUsername: strings.TrimSpace(input.RecipientUsername),
	})
}

// revokeInvite applies this package workflow transition.
func (s inviteMutationService) revokeInvite(ctx context.Context, campaignID string, input RevokeInviteInput) error {
	if err := s.auth.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	inviteID := strings.TrimSpace(input.InviteID)
	if inviteID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.invite_id_is_required", "invite id is required")
	}
	return s.mutation.RevokeInvite(ctx, campaignID, RevokeInviteInput{InviteID: inviteID})
}
