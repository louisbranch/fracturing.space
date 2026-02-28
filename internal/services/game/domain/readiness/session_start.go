package readiness

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

const (
	// RejectionCodeSessionReadinessGMRequired indicates there is no active GM participant.
	RejectionCodeSessionReadinessGMRequired = "SESSION_READINESS_GM_REQUIRED"
	// RejectionCodeSessionReadinessPlayerRequired indicates there is no active player participant.
	RejectionCodeSessionReadinessPlayerRequired = "SESSION_READINESS_PLAYER_REQUIRED"
	// RejectionCodeSessionReadinessPlayerCharacterRequired indicates a player controls no active characters.
	RejectionCodeSessionReadinessPlayerCharacterRequired = "SESSION_READINESS_PLAYER_CHARACTER_REQUIRED"
	// RejectionCodeSessionReadinessCharacterControllerRequired indicates an active character is unassigned.
	RejectionCodeSessionReadinessCharacterControllerRequired = "SESSION_READINESS_CHARACTER_CONTROLLER_REQUIRED"
	// RejectionCodeSessionReadinessCharacterSystemRequired indicates system-specific character readiness failed.
	RejectionCodeSessionReadinessCharacterSystemRequired = "SESSION_READINESS_CHARACTER_SYSTEM_REQUIRED"
)

// Rejection describes why session-start readiness evaluation failed.
type Rejection struct {
	Code    string
	Message string
}

// CharacterSystemReadiness checks optional system-specific readiness for a
// character's system profile payload.
//
// Returning false blocks session start. The reason should be concise and
// operator-readable because it is surfaced directly in domain rejection messages.
type CharacterSystemReadiness func(systemProfile map[string]any) (ready bool, reason string)

// EvaluateSessionStart evaluates campaign readiness invariants required before
// accepting session.start.
//
// The checks are deterministic and run only against aggregate replay state:
//  1. at least one active GM participant,
//  2. at least one active player participant,
//  3. each active player controls at least one active character,
//  4. every active character has a controller,
//  5. optional system-specific readiness for every active character.
func EvaluateSessionStart(state aggregate.State, systemReadiness CharacterSystemReadiness) *Rejection {
	activeParticipants := activeParticipantsByID(state)
	if len(activeParticipants.gmIDs) == 0 {
		return &Rejection{
			Code:    RejectionCodeSessionReadinessGMRequired,
			Message: "campaign readiness requires at least one gm participant",
		}
	}
	if len(activeParticipants.playerIDs) == 0 {
		return &Rejection{
			Code:    RejectionCodeSessionReadinessPlayerRequired,
			Message: "campaign readiness requires at least one player participant",
		}
	}

	activeCharacters := activeCharactersByID(state)
	playerCharacterCounts := make(map[string]int, len(activeParticipants.playerIDs))
	for _, playerID := range activeParticipants.playerIDs {
		playerCharacterCounts[playerID] = 0
	}

	for _, characterID := range activeCharacters.ids {
		characterState := activeCharacters.byID[characterID]
		controllerParticipantID := strings.TrimSpace(characterState.ParticipantID)
		if controllerParticipantID == "" {
			return &Rejection{
				Code:    RejectionCodeSessionReadinessCharacterControllerRequired,
				Message: fmt.Sprintf("campaign readiness requires character %s to have a controller", characterID),
			}
		}

		if _, ok := playerCharacterCounts[controllerParticipantID]; ok {
			playerCharacterCounts[controllerParticipantID]++
		}

		if systemReadiness == nil {
			continue
		}
		ready, reason := systemReadiness(characterState.SystemProfile)
		if ready {
			continue
		}
		message := fmt.Sprintf("campaign readiness requires character %s to satisfy system readiness", characterID)
		reason = strings.TrimSpace(reason)
		if reason != "" {
			message = message + ": " + reason
		}
		return &Rejection{
			Code:    RejectionCodeSessionReadinessCharacterSystemRequired,
			Message: message,
		}
	}

	for _, playerID := range activeParticipants.playerIDs {
		if playerCharacterCounts[playerID] > 0 {
			continue
		}
		return &Rejection{
			Code: RejectionCodeSessionReadinessPlayerCharacterRequired,
			Message: fmt.Sprintf(
				"campaign readiness requires player participant %s to control at least one character",
				playerID,
			),
		}
	}

	return nil
}

type participantIndex struct {
	byID      map[string]participant.State
	gmIDs     []string
	playerIDs []string
}

func activeParticipantsByID(state aggregate.State) participantIndex {
	indexed := participantIndex{byID: make(map[string]participant.State)}
	if len(state.Participants) == 0 {
		return indexed
	}

	ids := make([]string, 0, len(state.Participants))
	for participantID := range state.Participants {
		ids = append(ids, participantID)
	}
	sort.Strings(ids)

	for _, participantID := range ids {
		participantState := state.Participants[participantID]
		if !participantState.Joined || participantState.Left {
			continue
		}
		indexed.byID[participantID] = participantState
		role, ok := participant.NormalizeRole(participantState.Role)
		if !ok {
			continue
		}
		switch role {
		case participant.RoleGM:
			indexed.gmIDs = append(indexed.gmIDs, participantID)
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
	ParticipantID string
	SystemProfile map[string]any
}

func activeCharactersByID(state aggregate.State) characterIndex {
	indexed := characterIndex{byID: make(map[string]aggregateCharacterState)}
	if len(state.Characters) == 0 {
		return indexed
	}

	ids := make([]string, 0, len(state.Characters))
	for characterID := range state.Characters {
		ids = append(ids, characterID)
	}
	sort.Strings(ids)

	for _, characterID := range ids {
		characterState := state.Characters[characterID]
		if !characterState.Created || characterState.Deleted {
			continue
		}
		indexed.byID[characterID] = aggregateCharacterState{
			ParticipantID: characterState.ParticipantID,
			SystemProfile: characterState.SystemProfile,
		}
		indexed.ids = append(indexed.ids, characterID)
	}

	return indexed
}
