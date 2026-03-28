package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// SetCharacterOwner applies this package workflow transition.
func (s characterOwnershipService) SetCharacterOwner(ctx context.Context, campaignID string, characterID string, participantID string) error {
	return s.setCharacterOwner(ctx, campaignID, characterID, participantID)
}

// setCharacterOwner applies this package workflow transition.
func (s characterOwnershipService) setCharacterOwner(ctx context.Context, campaignID string, characterID string, participantID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyManageCharacter, characterID); err != nil {
		return err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return err
	}
	return s.mutation.SetCharacterOwner(ctx, campaignID, characterID, strings.TrimSpace(participantID))
}
