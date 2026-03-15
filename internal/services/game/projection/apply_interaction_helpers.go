package projection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func loadSessionInteraction(ctx context.Context, store storage.SessionInteractionStore, campaignID, sessionID string) (storage.SessionInteraction, error) {
	current, err := store.GetSessionInteraction(ctx, campaignID, sessionID)
	if err == nil {
		return current, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return storage.SessionInteraction{}, fmt.Errorf("get session interaction: %w", err)
	}
	return storage.SessionInteraction{
		CampaignID:                  campaignID,
		SessionID:                   sessionID,
		AITurn:                      storage.SessionAITurn{Status: session.AITurnStatusIdle},
		OOCPosts:                    []storage.SessionOOCPost{},
		ReadyToResumeParticipantIDs: []string{},
	}, nil
}

func loadSceneInteraction(ctx context.Context, store storage.SceneInteractionStore, campaignID, sceneID, sessionID string) (storage.SceneInteraction, error) {
	current, err := store.GetSceneInteraction(ctx, campaignID, sceneID)
	if err == nil {
		if strings.TrimSpace(current.SessionID) == "" {
			current.SessionID = strings.TrimSpace(sessionID)
		}
		return current, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return storage.SceneInteraction{}, fmt.Errorf("get scene interaction: %w", err)
	}
	return storage.SceneInteraction{
		CampaignID:           campaignID,
		SceneID:              sceneID,
		SessionID:            strings.TrimSpace(sessionID),
		ActingCharacterIDs:   []string{},
		ActingParticipantIDs: []string{},
		Slots:                []storage.ScenePlayerSlot{},
	}, nil
}

func characterIDsToStrings(values []ids.CharacterID) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value.String()); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func participantIDsToStrings(values []ids.ParticipantID) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value.String()); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func newScenePlayerSlots(participantIDs []string) []storage.ScenePlayerSlot {
	if len(participantIDs) == 0 {
		return []storage.ScenePlayerSlot{}
	}
	slots := make([]storage.ScenePlayerSlot, 0, len(participantIDs))
	for _, participantID := range participantIDs {
		participantID = strings.TrimSpace(participantID)
		if participantID == "" {
			continue
		}
		slots = append(slots, storage.ScenePlayerSlot{
			ParticipantID:      participantID,
			ReviewStatus:       scene.PlayerPhaseSlotReviewStatusOpen,
			CharacterIDs:       []string{},
			ReviewCharacterIDs: []string{},
		})
	}
	return slots
}

func findOrCreateScenePlayerSlot(slots []storage.ScenePlayerSlot, participantID string) storage.ScenePlayerSlot {
	participantID = strings.TrimSpace(participantID)
	for i := range slots {
		if slots[i].ParticipantID == participantID {
			return slots[i]
		}
	}
	return storage.ScenePlayerSlot{
		ParticipantID:      participantID,
		CharacterIDs:       []string{},
		ReviewStatus:       scene.PlayerPhaseSlotReviewStatusOpen,
		ReviewCharacterIDs: []string{},
	}
}

func upsertScenePlayerSlot(slots []storage.ScenePlayerSlot, updated storage.ScenePlayerSlot) []storage.ScenePlayerSlot {
	for i := range slots {
		if slots[i].ParticipantID == updated.ParticipantID {
			slots[i] = updated
			return slots
		}
	}
	return append(slots, updated)
}

func deleteString(values []string, target string) []string {
	if target == "" || len(values) == 0 {
		return values
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
