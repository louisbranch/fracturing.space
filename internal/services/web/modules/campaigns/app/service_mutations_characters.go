package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

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
	if err := s.requirePolicy(ctx, campaignID, policyMutateCharacter); err != nil {
		return err
	}
	return s.mutationGateway.UpdateCharacter(ctx, campaignID, characterID, UpdateCharacterInput{
		Name:     name,
		Pronouns: strings.TrimSpace(input.Pronouns),
	})
}
