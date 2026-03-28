package readiness

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
)

func evaluateCoreSessionStartBlockers(state aggregate.State, systemReadiness CharacterSystemReadiness) []Blocker {
	blockers := make([]Blocker, 0, 4)
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
	for _, characterID := range activeCharacters.ids {
		characterState := activeCharacters.byID[characterID]
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
			completeCharacterAction(activeParticipants, strings.TrimSpace(characterState.OwnerParticipantID), characterID),
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
