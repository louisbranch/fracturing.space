package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// SetCharacterController applies this package workflow transition.
func (s characterControlService) SetCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	return s.setCharacterController(ctx, campaignID, characterID, participantID)
}

// ClaimCharacterControl applies this package workflow transition.
func (s characterControlService) ClaimCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	return s.claimCharacterControl(ctx, campaignID, characterID, userID)
}

// ReleaseCharacterControl applies this package workflow transition.
func (s characterControlService) ReleaseCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	return s.releaseCharacterControl(ctx, campaignID, characterID, userID)
}

// setCharacterController applies this package workflow transition.
func (s characterControlService) setCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
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
	return s.mutation.SetCharacterController(ctx, campaignID, characterID, strings.TrimSpace(participantID))
}

// claimCharacterControl applies this package workflow transition.
func (s characterControlService) claimCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if _, err := s.requireViewerCampaignParticipant(ctx, campaignID, userID); err != nil {
		return err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return err
	}
	return s.mutation.ClaimCharacterControl(ctx, campaignID, characterID)
}

// releaseCharacterControl applies this package workflow transition.
func (s characterControlService) releaseCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if _, err := s.requireViewerCampaignParticipant(ctx, campaignID, userID); err != nil {
		return err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return err
	}
	return s.mutation.ReleaseCharacterControl(ctx, campaignID, characterID)
}

// requireViewerCampaignParticipant ensures the current viewer is linked to a
// campaign participant before self-service character control changes run.
func (s characterControlService) requireViewerCampaignParticipant(ctx context.Context, campaignID string, userID string) (CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignParticipant{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return CampaignParticipant{}, apperrors.EK(apperrors.KindForbidden, policyMutateCharacter.denyKey, policyMutateCharacter.denyMsg)
	}
	participants, err := characterParticipants(ctx, s.participants, campaignID)
	if err != nil {
		return CampaignParticipant{}, err
	}
	participant := campaignParticipantForUserID(participants, userID)
	if strings.TrimSpace(participant.ID) == "" {
		return CampaignParticipant{}, apperrors.EK(apperrors.KindForbidden, policyMutateCharacter.denyKey, policyMutateCharacter.denyMsg)
	}
	return participant, nil
}
