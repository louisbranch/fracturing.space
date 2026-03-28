package readiness

import (
	"sort"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	domainids "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

type participantIndex struct {
	byID      map[string]participant.State
	gmIDs     []string
	aiGMIDs   []string
	playerIDs []string
}

func activeParticipantsByID(state aggregate.State) participantIndex {
	indexed := participantIndex{byID: make(map[string]participant.State)}
	if len(state.Participants) == 0 {
		return indexed
	}

	pids := make([]string, 0, len(state.Participants))
	for participantID := range state.Participants {
		pids = append(pids, string(participantID))
	}
	sort.Strings(pids)

	for _, participantID := range pids {
		participantState := state.Participants[domainids.ParticipantID(participantID)]
		if !participantState.Joined || participantState.Left {
			continue
		}
		indexed.byID[participantID] = participantState
		role, ok := participant.NormalizeRole(string(participantState.Role))
		if !ok {
			continue
		}
		switch role {
		case participant.RoleGM:
			indexed.gmIDs = append(indexed.gmIDs, participantID)
			controller, ok := participant.NormalizeController(string(participantState.Controller))
			if ok && controller == participant.ControllerAI {
				indexed.aiGMIDs = append(indexed.aiGMIDs, participantID)
			}
		case participant.RolePlayer:
			indexed.playerIDs = append(indexed.playerIDs, participantID)
		}
	}

	return indexed
}

type characterIndex struct {
	byID map[string]aggregateCharacterState
	ids  []string
}

type aggregateCharacterState struct {
	OwnerParticipantID string
	Name               string
}

func activeCharactersByID(state aggregate.State) characterIndex {
	indexed := characterIndex{byID: make(map[string]aggregateCharacterState)}
	if len(state.Characters) == 0 {
		return indexed
	}

	cids := make([]string, 0, len(state.Characters))
	for characterID := range state.Characters {
		cids = append(cids, string(characterID))
	}
	sort.Strings(cids)

	for _, characterID := range cids {
		characterState := state.Characters[domainids.CharacterID(characterID)]
		if !characterState.Created || characterState.Deleted {
			continue
		}
		indexed.byID[characterID] = aggregateCharacterState{
			OwnerParticipantID: string(characterState.OwnerParticipantID),
			Name:               characterState.Name,
		}
		indexed.ids = append(indexed.ids, characterID)
	}

	return indexed
}
