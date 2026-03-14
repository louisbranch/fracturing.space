package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// requireNoActiveSessionForCharacterMutation blocks character create/update
// flows while gameplay is active.
func (s service) requireNoActiveSessionForCharacterMutation(ctx context.Context, campaignID string) error {
	sessions, err := s.readGateway.CampaignSessions(ctx, campaignID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if strings.EqualFold(strings.TrimSpace(session.Status), "active") {
			return apperrors.EK(
				apperrors.KindConflict,
				"error.web.message.active_session_blocks_character_mutation",
				"active session blocks character mutation",
			)
		}
	}
	return nil
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
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
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

// updateCharacter applies this package workflow transition.
func (s service) updateCharacter(ctx context.Context, campaignID string, characterID string, input UpdateCharacterInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
		return err
	}
	return s.mutationGateway.UpdateCharacter(ctx, campaignID, characterID, UpdateCharacterInput{
		Name:     name,
		Pronouns: strings.TrimSpace(input.Pronouns),
	})
}

// deleteCharacter applies this package workflow transition.
func (s service) deleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
		return err
	}
	return s.mutationGateway.DeleteCharacter(ctx, campaignID, characterID)
}

// setCharacterController applies this package workflow transition.
func (s service) setCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyManageCharacter, characterID); err != nil {
		return err
	}
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
		return err
	}
	return s.mutationGateway.SetCharacterController(ctx, campaignID, characterID, strings.TrimSpace(participantID))
}

// claimCharacterControl applies this package workflow transition.
func (s service) claimCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
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
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
		return err
	}
	return s.mutationGateway.ClaimCharacterControl(ctx, campaignID, characterID)
}

// releaseCharacterControl applies this package workflow transition.
func (s service) releaseCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
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
	if err := s.requireNoActiveSessionForCharacterMutation(ctx, campaignID); err != nil {
		return err
	}
	return s.mutationGateway.ReleaseCharacterControl(ctx, campaignID, characterID)
}

// requireViewerCampaignParticipant ensures the current viewer is linked to a
// campaign participant before self-service character control changes run.
func (s service) requireViewerCampaignParticipant(ctx context.Context, campaignID string, userID string) (CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignParticipant{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return CampaignParticipant{}, apperrors.EK(apperrors.KindForbidden, policyMutateCharacter.denyKey, policyMutateCharacter.denyMsg)
	}
	participants, err := s.campaignParticipants(ctx, campaignID)
	if err != nil {
		return CampaignParticipant{}, err
	}
	participant := campaignParticipantForUserID(participants, userID)
	if strings.TrimSpace(participant.ID) == "" {
		return CampaignParticipant{}, apperrors.EK(apperrors.KindForbidden, policyMutateCharacter.denyKey, policyMutateCharacter.denyMsg)
	}
	return participant, nil
}
