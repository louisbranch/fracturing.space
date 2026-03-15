package readiness

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func newBlocker(code, message string, metadata map[string]string) Blocker {
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return Blocker{
		Code:     code,
		Message:  message,
		Metadata: cloned,
		Action:   Action{},
	}
}

func newActionableBlocker(code, message string, metadata map[string]string, action Action) Blocker {
	blocker := newBlocker(code, message, metadata)
	blocker.Action = cloneAction(action)
	return blocker
}

func aiAgentRequiredAction(activeParticipants participantIndex) Action {
	action := ownerManageParticipantsAction(activeParticipants)
	action.ResolutionKind = ResolutionKindConfigureAIAgent
	if len(activeParticipants.aiGMIDs) > 0 {
		action.TargetParticipantID = activeParticipants.aiGMIDs[0]
	}
	return action
}

func ownerManageParticipantsAction(activeParticipants participantIndex) Action {
	ownerIDs, ownerUserIDs := ownerParticipants(activeParticipants)
	return Action{
		ResponsibleUserIDs:        ownerUserIDs,
		ResponsibleParticipantIDs: ownerIDs,
		ResolutionKind:            ResolutionKindManageParticipants,
	}
}

func invitePlayerAction(activeParticipants participantIndex) Action {
	responsibleParticipants := make([]string, 0, len(activeParticipants.gmIDs))
	responsibleUsers := make([]string, 0, len(activeParticipants.gmIDs))
	for _, participantID := range activeParticipants.gmIDs {
		state, ok := activeParticipants.byID[participantID]
		if !ok {
			continue
		}
		controller, controllerOK := participant.NormalizeController(string(state.Controller))
		if controllerOK && controller == participant.ControllerAI {
			continue
		}
		if userID := strings.TrimSpace(string(state.UserID)); userID != "" {
			responsibleParticipants = append(responsibleParticipants, participantID)
			responsibleUsers = append(responsibleUsers, userID)
		}
	}
	return Action{
		ResponsibleUserIDs:        responsibleUsers,
		ResponsibleParticipantIDs: responsibleParticipants,
		ResolutionKind:            ResolutionKindInvitePlayer,
	}
}

func createCharacterAction(activeParticipants participantIndex, participantID string) Action {
	participantID = strings.TrimSpace(participantID)
	state, ok := activeParticipants.byID[participantID]
	if !ok {
		return Action{}
	}
	userID := strings.TrimSpace(string(state.UserID))
	if userID == "" {
		return Action{}
	}
	return Action{
		ResponsibleUserIDs:        []string{userID},
		ResponsibleParticipantIDs: []string{participantID},
		ResolutionKind:            ResolutionKindCreateCharacter,
		TargetParticipantID:       participantID,
	}
}

func completeCharacterAction(activeParticipants participantIndex, participantID, characterID string) Action {
	participantID = strings.TrimSpace(participantID)
	state, ok := activeParticipants.byID[participantID]
	if !ok {
		return Action{}
	}
	userID := strings.TrimSpace(string(state.UserID))
	if userID == "" {
		return Action{}
	}
	return Action{
		ResponsibleUserIDs:        []string{userID},
		ResponsibleParticipantIDs: []string{participantID},
		ResolutionKind:            ResolutionKindCompleteCharacter,
		TargetParticipantID:       participantID,
		TargetCharacterID:         strings.TrimSpace(characterID),
	}
}

func ownerParticipants(activeParticipants participantIndex) ([]string, []string) {
	participantIDs := make([]string, 0, len(activeParticipants.byID))
	userIDs := make([]string, 0, len(activeParticipants.byID))
	for _, candidateID := range append(append([]string{}, activeParticipants.gmIDs...), activeParticipants.playerIDs...) {
		state, ok := activeParticipants.byID[candidateID]
		if !ok {
			continue
		}
		access, accessOK := participant.NormalizeCampaignAccess(string(state.CampaignAccess))
		if !accessOK || access != participant.CampaignAccessOwner {
			continue
		}
		if userID := strings.TrimSpace(string(state.UserID)); userID != "" {
			participantIDs = append(participantIDs, candidateID)
			userIDs = append(userIDs, userID)
		}
	}
	return normalizeActionIDs(participantIDs), normalizeActionIDs(userIDs)
}
