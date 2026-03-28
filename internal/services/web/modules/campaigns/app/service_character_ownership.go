package app

import (
	"context"
	"strings"
)

// CampaignCharacterOwnership centralizes this web behavior in one helper seam.
func (s characterOwnershipService) CampaignCharacterOwnership(ctx context.Context, campaignID string, characterID string, options CharacterReadContext) (CampaignCharacterOwnership, error) {
	return s.campaignCharacterOwnership(ctx, campaignID, characterID, options)
}

// campaignCharacterOwnership centralizes character-detail ownership state.
func (s characterOwnershipService) campaignCharacterOwnership(ctx context.Context, campaignID string, characterID string, options CharacterReadContext) (CampaignCharacterOwnership, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterOwnership{}, nil
	}

	character, err := loadCharacterForOwnership(ctx, s.read, s.auth.gateway, campaignID, characterID, options)
	if err != nil {
		return CampaignCharacterOwnership{}, err
	}
	if strings.TrimSpace(character.ID) == "" {
		return CampaignCharacterOwnership{}, nil
	}

	participants, err := characterParticipants(ctx, s.participants, campaignID)
	if err != nil {
		return CampaignCharacterOwnership{}, err
	}

	ownership := CampaignCharacterOwnership{
		CurrentOwnerName: strings.TrimSpace(character.Owner),
	}
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyManageCharacter, characterID); err == nil {
		ownership.CanManageOwnership = true
		ownership.Options = campaignCharacterOwnershipOptions(participants, character.OwnerParticipantID)
	}

	return ownership, nil
}

// loadCharacterForOwnership resolves one character plus editability state for
// detail/ownership flows without depending on the broader read service type.
func loadCharacterForOwnership(
	ctx context.Context,
	read CampaignCharacterReadGateway,
	auth AuthorizationGateway,
	campaignID, characterID string,
	options CharacterReadContext,
) (CampaignCharacter, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacter{}, nil
	}

	character, err := read.CampaignCharacter(ctx, campaignID, characterID, options)
	if err != nil {
		return CampaignCharacter{}, err
	}
	normalized := normalizeCampaignCharacter(character)
	if strings.TrimSpace(normalized.ID) == "" {
		return CampaignCharacter{}, nil
	}
	if auth != nil {
		decision, err := auth.CanCampaignAction(
			ctx,
			campaignID,
			campaignAuthzActionMutate,
			campaignAuthzResourceCharacter,
			&AuthorizationTarget{ResourceID: normalized.ID},
		)
		if err == nil {
			normalized.EditReasonCode = strings.TrimSpace(decision.ReasonCode)
			normalized.CanEdit = decision.Evaluated && decision.Allowed
		}
	}
	return normalized, nil
}

// characterParticipants loads the participant roster used by character-owner
// flows without routing through the participant service seam.
func characterParticipants(ctx context.Context, participants CampaignParticipantReadGateway, campaignID string) ([]CampaignParticipant, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignParticipant{}, nil
	}

	items, err := participants.CampaignParticipants(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []CampaignParticipant{}, nil
	}

	normalized := make([]CampaignParticipant, 0, len(items))
	for _, participant := range items {
		normalized = append(normalized, normalizeCampaignParticipant(participant))
	}
	sortByName(normalized, func(p CampaignParticipant) string { return p.Name }, func(p CampaignParticipant) string { return p.ID })
	return normalized, nil
}

// campaignCharacterOwnershipOptions builds the owner selector state for the
// character-detail ownership form.
func campaignCharacterOwnershipOptions(participants []CampaignParticipant, selectedParticipantID string) []CampaignCharacterOwnershipOption {
	selectedParticipantID = strings.TrimSpace(selectedParticipantID)
	options := []CampaignCharacterOwnershipOption{{
		ParticipantID: "",
		Label:         "Unassigned",
		Selected:      selectedParticipantID == "",
	}}
	for _, participant := range participants {
		participantID := strings.TrimSpace(participant.ID)
		if participantID == "" {
			continue
		}
		label := strings.TrimSpace(participant.Name)
		if label == "" {
			label = participantID
		}
		options = append(options, CampaignCharacterOwnershipOption{
			ParticipantID: participantID,
			Label:         label,
			Selected:      participantID == selectedParticipantID,
		})
	}
	return options
}
