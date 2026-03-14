package app

import (
	"context"
	"strings"
)

// campaignCharacters centralizes this web behavior in one helper seam.
func (s service) campaignCharacters(ctx context.Context, campaignID string, options CampaignCharactersReadOptions) ([]CampaignCharacter, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignCharacter{}, nil
	}

	characters, err := s.readGateway.CampaignCharacters(ctx, campaignID, options)
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
			ID:                      characterID,
			Name:                    characterName,
			Kind:                    kind,
			Controller:              controller,
			ControllerParticipantID: strings.TrimSpace(character.ControllerParticipantID),
			Pronouns:                strings.TrimSpace(character.Pronouns),
			Aliases:                 append([]string(nil), character.Aliases...),
			AvatarURL:               strings.TrimSpace(character.AvatarURL),
			Daggerheart:             normalizeCampaignCharacterDaggerheartSummary(character.Daggerheart),
		})
	}

	sortByName(normalized, func(c CampaignCharacter) string { return c.Name }, func(c CampaignCharacter) string { return c.ID })

	s.hydrateCharacterEditability(ctx, campaignID, normalized)

	return normalized, nil
}

// campaignCharacter centralizes this web behavior in one helper seam.
func (s service) campaignCharacter(ctx context.Context, campaignID string, characterID string) (CampaignCharacter, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacter{}, nil
	}

	characters, err := s.campaignCharacters(ctx, campaignID, CampaignCharactersReadOptions{})
	if err != nil {
		return CampaignCharacter{}, err
	}
	for _, character := range characters {
		if strings.TrimSpace(character.ID) == characterID {
			return character, nil
		}
	}
	return CampaignCharacter{}, nil
}

// campaignCharacterEditor centralizes this web behavior in one helper seam.
func (s service) campaignCharacterEditor(ctx context.Context, campaignID string, characterID string) (CampaignCharacterEditor, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterEditor{}, nil
	}

	character, err := s.campaignCharacter(ctx, campaignID, characterID)
	if err != nil {
		return CampaignCharacterEditor{}, err
	}
	if strings.TrimSpace(character.ID) == "" {
		return CampaignCharacterEditor{}, nil
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return CampaignCharacterEditor{}, err
	}
	return CampaignCharacterEditor{Character: character}, nil
}

// hydrateCharacterEditability centralizes this web behavior in one helper seam.
func (s service) hydrateCharacterEditability(ctx context.Context, campaignID string, characters []CampaignCharacter) {
	if len(characters) == 0 {
		return
	}
	if s.authzGateway == nil {
		return
	}

	checks, indexesByCheckID := buildAuthorizationChecksByID(
		len(characters),
		func(idx int) string { return characters[idx].ID },
		func(checkID string, _ int) AuthorizationCheck {
			return AuthorizationCheck{
				CheckID:  checkID,
				Action:   campaignAuthzActionMutate,
				Resource: campaignAuthzResourceCharacter,
				Target: &AuthorizationTarget{
					ResourceID: checkID,
				},
			}
		},
	)
	if len(checks) == 0 {
		return
	}

	decisions, err := s.authzGateway.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return
	}

	applyAuthorizationDecisions(checks, indexesByCheckID, decisions, func(characterIndex int, decision AuthorizationDecision) {
		characters[characterIndex].EditReasonCode = strings.TrimSpace(decision.ReasonCode)
		if decision.Evaluated && decision.Allowed {
			characters[characterIndex].CanEdit = true
		}
	})
}

// normalizeCampaignCharacterDaggerheartSummary drops partial Daggerheart card
// summaries so transport and templates never render unresolved catalog IDs.
func normalizeCampaignCharacterDaggerheartSummary(summary *CampaignCharacterDaggerheartSummary) *CampaignCharacterDaggerheartSummary {
	if summary == nil {
		return nil
	}
	if summary.Level <= 0 {
		return nil
	}
	normalized := &CampaignCharacterDaggerheartSummary{
		Level:         summary.Level,
		ClassName:     strings.TrimSpace(summary.ClassName),
		SubclassName:  strings.TrimSpace(summary.SubclassName),
		AncestryName:  strings.TrimSpace(summary.AncestryName),
		CommunityName: strings.TrimSpace(summary.CommunityName),
	}
	if normalized.ClassName == "" || normalized.SubclassName == "" || normalized.AncestryName == "" || normalized.CommunityName == "" {
		return nil
	}
	return normalized
}
