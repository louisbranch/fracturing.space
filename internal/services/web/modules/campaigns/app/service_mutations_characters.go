package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CreateCharacter executes package-scoped creation behavior for this flow.
func (s characterMutationService) CreateCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	return s.createCharacter(ctx, campaignID, input)
}

// UpdateCharacter applies this package workflow transition.
func (s characterMutationService) UpdateCharacter(ctx context.Context, campaignID string, characterID string, input UpdateCharacterInput) error {
	return s.updateCharacter(ctx, campaignID, characterID, input)
}

// DeleteCharacter applies this package workflow transition.
func (s characterMutationService) DeleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	return s.deleteCharacter(ctx, campaignID, characterID)
}

// requireNoActiveSessionForCharacterMutation blocks character create/update
// flows while gameplay is active.
func requireNoActiveSessionForCharacterMutation(ctx context.Context, sessions CampaignSessionReadGateway, campaignID string) error {
	items, err := sessions.CampaignSessions(ctx, campaignID)
	if err != nil {
		return err
	}
	for _, session := range items {
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
func (s characterMutationService) createCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	if input.Kind == CharacterKindUnspecified {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}
	if err := s.auth.requirePolicy(ctx, campaignID, policyMutateCharacter); err != nil {
		return CreateCharacterResult{}, err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return CreateCharacterResult{}, err
	}
	created, err := s.mutation.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		return CreateCharacterResult{}, err
	}
	if strings.TrimSpace(created.CharacterID) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_character_id_was_empty", "created character id was empty")
	}
	return created, nil
}

// updateCharacter applies this package workflow transition.
func (s characterMutationService) updateCharacter(ctx context.Context, campaignID string, characterID string, input UpdateCharacterInput) error {
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
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return err
	}
	return s.mutation.UpdateCharacter(ctx, campaignID, characterID, UpdateCharacterInput{
		Name:     name,
		Pronouns: strings.TrimSpace(input.Pronouns),
	})
}

// deleteCharacter applies this package workflow transition.
func (s characterMutationService) deleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	if err := requireNoActiveSessionForCharacterMutation(ctx, s.sessions, campaignID); err != nil {
		return err
	}
	return s.mutation.DeleteCharacter(ctx, campaignID, characterID)
}
