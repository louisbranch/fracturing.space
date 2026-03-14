package app

import (
	"context"
	"strings"
)

// CampaignCharacterControl centralizes this web behavior in one helper seam.
func (s characterControlService) CampaignCharacterControl(ctx context.Context, campaignID string, characterID string, userID string, options CharacterReadContext) (CampaignCharacterControl, error) {
	return s.campaignCharacterControl(ctx, campaignID, characterID, userID, options)
}

// campaignCharacterControl centralizes character-detail control state.
func (s characterControlService) campaignCharacterControl(ctx context.Context, campaignID string, characterID string, userID string, options CharacterReadContext) (CampaignCharacterControl, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterControl{}, nil
	}

	character, err := loadCharacterForControl(ctx, s.read, s.auth.gateway, campaignID, characterID, options)
	if err != nil {
		return CampaignCharacterControl{}, err
	}
	if strings.TrimSpace(character.ID) == "" {
		return CampaignCharacterControl{}, nil
	}

	participants, err := characterParticipants(ctx, s.participants, campaignID)
	if err != nil {
		return CampaignCharacterControl{}, err
	}

	control := CampaignCharacterControl{}
	currentParticipant := campaignParticipantForUserID(participants, userID)
	if strings.TrimSpace(currentParticipant.ID) != "" {
		control.CurrentParticipantName = strings.TrimSpace(currentParticipant.Name)
		switch controllerID := strings.TrimSpace(character.ControllerParticipantID); {
		case controllerID == "":
			control.CanSelfClaim = true
		case controllerID == strings.TrimSpace(currentParticipant.ID):
			control.CanSelfRelease = true
		}
	}

	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyManageCharacter, characterID); err == nil {
		control.CanManageControl = true
		control.Options = campaignCharacterControlOptions(participants, character.ControllerParticipantID)
	}

	return control, nil
}

// loadCharacterForControl resolves one character plus editability state for
// detail/control flows without depending on the broader read service type.
func loadCharacterForControl(
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

// campaignParticipantForUserID finds the participant seat linked to the
// current authenticated user for this campaign workspace.
func campaignParticipantForUserID(participants []CampaignParticipant, userID string) CampaignParticipant {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return CampaignParticipant{}
	}
	for _, participant := range participants {
		if strings.TrimSpace(participant.UserID) == userID {
			return participant
		}
	}
	return CampaignParticipant{}
}

// characterParticipants loads the participant roster used by character-control
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

// campaignCharacterControlOptions builds the manager override selector state
// for the character-detail control form.
func campaignCharacterControlOptions(participants []CampaignParticipant, selectedParticipantID string) []CampaignCharacterControlOption {
	selectedParticipantID = strings.TrimSpace(selectedParticipantID)
	options := []CampaignCharacterControlOption{{
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
		options = append(options, CampaignCharacterControlOption{
			ParticipantID: participantID,
			Label:         label,
			Selected:      participantID == selectedParticipantID,
		})
	}
	return options
}
