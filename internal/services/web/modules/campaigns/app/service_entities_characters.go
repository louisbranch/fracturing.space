package app

import (
	"context"
	"strings"
)

// campaignCharacters centralizes this web behavior in one helper seam.
func (s service) campaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignCharacter{}, nil
	}

	characters, err := s.readGateway.CampaignCharacters(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return []CampaignCharacter{}, nil
	}

	normalized := make([]CampaignCharacter, 0, len(characters))
	for _, character := range characters {
		characterID := strings.TrimSpace(character.ID)
		characterName := strings.TrimSpace(character.Name)
		if characterName == "" {
			if characterID != "" {
				characterName = characterID
			} else {
				characterName = "Unknown character"
			}
		}
		kind := strings.TrimSpace(character.Kind)
		if kind == "" {
			kind = "Unspecified"
		}
		controller := strings.TrimSpace(character.Controller)
		if controller == "" {
			controller = "Unassigned"
		}
		normalized = append(normalized, CampaignCharacter{
			ID:         characterID,
			Name:       characterName,
			Kind:       kind,
			Controller: controller,
			Pronouns:   strings.TrimSpace(character.Pronouns),
			Aliases:    append([]string(nil), character.Aliases...),
			AvatarURL:  strings.TrimSpace(character.AvatarURL),
		})
	}

	sortByName(normalized, func(c CampaignCharacter) string { return c.Name }, func(c CampaignCharacter) string { return c.ID })

	s.hydrateCharacterEditability(ctx, campaignID, normalized)

	return normalized, nil
}

// hydrateCharacterEditability centralizes this web behavior in one helper seam.
func (s service) hydrateCharacterEditability(ctx context.Context, campaignID string, characters []CampaignCharacter) {
	if len(characters) == 0 {
		return
	}
	if s.authzGateway == nil {
		return
	}

	checks := make([]AuthorizationCheck, 0, len(characters))
	indexesByCheckID := make(map[string][]int, len(characters))
	for idx := range characters {
		characterID := strings.TrimSpace(characters[idx].ID)
		if characterID == "" {
			continue
		}
		indexesByCheckID[characterID] = append(indexesByCheckID[characterID], idx)
		if len(indexesByCheckID[characterID]) > 1 {
			continue
		}
		checks = append(checks, AuthorizationCheck{
			CheckID:  characterID,
			Action:   campaignAuthzActionMutate,
			Resource: campaignAuthzResourceCharacter,
			Target: &AuthorizationTarget{
				ResourceID: characterID,
			},
		})
	}
	if len(checks) == 0 {
		return
	}

	decisions, err := s.authzGateway.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return
	}

	for idx, decision := range decisions {
		checkID := strings.TrimSpace(decision.CheckID)
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		if checkID == "" {
			continue
		}
		characterIndexes, found := indexesByCheckID[checkID]
		if !found {
			continue
		}
		for _, characterIndex := range characterIndexes {
			characters[characterIndex].EditReasonCode = strings.TrimSpace(decision.ReasonCode)
			if decision.Evaluated && decision.Allowed {
				characters[characterIndex].CanEdit = true
			}
		}
	}
}
