package readiness

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
)

func evaluateCoreSessionStartBlockers(state aggregate.State, systemReadiness CharacterSystemReadiness) []Blocker {
	blockers := make([]Blocker, 0, 6)
	activeParticipants := activeParticipantsByID(state)

	if isAIGMMode(state.Campaign.GmMode) && strings.TrimSpace(state.Campaign.AIAgentID) == "" {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessAIAgentRequired,
			"campaign readiness requires ai agent binding for ai gm mode",
			nil,
			aiAgentRequiredAction(activeParticipants),
		))
	}

	if isAIGMMode(state.Campaign.GmMode) && len(activeParticipants.aiGMIDs) == 0 {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessAIGMParticipantRequired,
			"campaign readiness requires at least one ai-controlled gm participant for ai gm mode",
			nil,
			ownerManageParticipantsAction(activeParticipants),
		))
	}
	if len(activeParticipants.gmIDs) == 0 {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessGMRequired,
			"campaign readiness requires at least one gm participant",
			nil,
			ownerManageParticipantsAction(activeParticipants),
		))
	}
	if len(activeParticipants.playerIDs) == 0 {
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessPlayerRequired,
			"campaign readiness requires at least one player participant",
			nil,
			invitePlayerAction(activeParticipants),
		))
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
			characterLabel, metadata := readinessCharacterLabelAndMetadata(characterID, characterState.Name)
			blockers = append(blockers, newBlocker(
				RejectionCodeSessionReadinessCharacterControllerRequired,
				fmt.Sprintf("campaign readiness requires character %s to have a controller", characterLabel),
				metadata,
			))
		}

		if _, ok := playerCharacterCounts[controllerParticipantID]; ok {
			playerCharacterCounts[controllerParticipantID]++
		}

		if systemReadiness == nil {
			continue
		}
		ready, reason := systemReadiness(characterID)
		if ready {
			continue
		}
		reason = strings.TrimSpace(reason)
		characterLabel, metadata := readinessCharacterLabelAndMetadata(characterID, characterState.Name)
		if reason != "" {
			metadata["reason"] = reason
		}
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessCharacterSystemRequired,
			systemReadinessMessage(characterLabel, reason),
			metadata,
			completeCharacterAction(activeParticipants, controllerParticipantID, characterID),
		))
	}

	for _, playerID := range activeParticipants.playerIDs {
		if playerCharacterCounts[playerID] > 0 {
			continue
		}
		playerState := activeParticipants.byID[playerID]
		playerName := strings.TrimSpace(playerState.Name)
		playerLabel := playerName
		if playerLabel == "" {
			playerLabel = playerID
		}
		metadata := map[string]string{
			"participant_id": playerID,
		}
		if playerName != "" {
			metadata["participant_name"] = playerName
		}
		blockers = append(blockers, newActionableBlocker(
			RejectionCodeSessionReadinessPlayerCharacterRequired,
			fmt.Sprintf("campaign readiness requires player participant %s to control at least one character", playerLabel),
			metadata,
			createCharacterAction(activeParticipants, playerID),
		))
	}

	return blockers
}

func readinessCharacterLabelAndMetadata(characterID, characterName string) (string, map[string]string) {
	trimmedID := strings.TrimSpace(characterID)
	metadata := map[string]string{
		"character_id": trimmedID,
	}

	trimmedName := strings.TrimSpace(characterName)
	if trimmedName != "" {
		metadata["character_name"] = trimmedName
		return trimmedName, metadata
	}

	return trimmedID, metadata
}

func systemReadinessMessage(characterLabel, reason string) string {
	message := fmt.Sprintf("campaign readiness requires character %s to satisfy system readiness", characterLabel)
	if reason == "" {
		return message
	}
	return message + ": " + reason
}
