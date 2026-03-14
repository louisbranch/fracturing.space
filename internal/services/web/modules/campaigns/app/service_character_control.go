package app

import (
	"context"
	"strings"
)

// campaignCharacterControl centralizes character-detail control state.
func (s service) campaignCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) (CampaignCharacterControl, error) {
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return CampaignCharacterControl{}, nil
	}

	character, err := s.campaignCharacter(ctx, campaignID, characterID)
	if err != nil {
		return CampaignCharacterControl{}, err
	}
	if strings.TrimSpace(character.ID) == "" {
		return CampaignCharacterControl{}, nil
	}

	participants, err := s.campaignParticipants(ctx, campaignID)
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

	if err := s.requirePolicyWithTarget(ctx, campaignID, policyManageCharacter, characterID); err == nil {
		control.CanManageControl = true
		control.Options = campaignCharacterControlOptions(participants, character.ControllerParticipantID)
	}

	return control, nil
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
