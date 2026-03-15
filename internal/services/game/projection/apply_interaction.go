package projection

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionActiveSceneSet(ctx context.Context, evt event.Event, payload session.ActiveSceneSetPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.ActiveSceneID = strings.TrimSpace(payload.ActiveSceneID.String())
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionGMAuthoritySet(ctx context.Context, evt event.Event, payload session.GMAuthoritySetPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.GMAuthorityParticipantID = strings.TrimSpace(payload.ParticipantID.String())
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCPaused(ctx context.Context, evt event.Event, _ session.OOCPausedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.OOCPaused = true
	current.OOCPosts = []storage.SessionOOCPost{}
	current.ReadyToResumeParticipantIDs = []string{}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCPosted(ctx context.Context, evt event.Event, payload session.OOCPostedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.OOCPosts = append(current.OOCPosts, storage.SessionOOCPost{
		PostID:        strings.TrimSpace(payload.PostID),
		ParticipantID: strings.TrimSpace(payload.ParticipantID.String()),
		Body:          strings.TrimSpace(payload.Body),
		CreatedAt:     updatedAt,
	})
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCReadyMarked(ctx context.Context, evt event.Event, payload session.OOCReadyMarkedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID != "" && !slices.Contains(current.ReadyToResumeParticipantIDs, participantID) {
		current.ReadyToResumeParticipantIDs = append(current.ReadyToResumeParticipantIDs, participantID)
	}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCReadyCleared(ctx context.Context, evt event.Event, payload session.OOCReadyClearedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	current.ReadyToResumeParticipantIDs = deleteString(current.ReadyToResumeParticipantIDs, participantID)
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCResumed(ctx context.Context, evt event.Event, _ session.OOCResumedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.OOCPaused = false
	current.OOCPosts = []storage.SessionOOCPost{}
	current.ReadyToResumeParticipantIDs = []string{}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnQueued(ctx context.Context, evt event.Event, payload session.AITurnQueuedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn = storage.SessionAITurn{
		Status:             session.AITurnStatusQueued,
		TurnToken:          strings.TrimSpace(payload.TurnToken),
		OwnerParticipantID: strings.TrimSpace(payload.OwnerParticipantID.String()),
		SourceEventType:    strings.TrimSpace(payload.SourceEventType),
		SourceSceneID:      strings.TrimSpace(payload.SourceSceneID.String()),
		SourcePhaseID:      strings.TrimSpace(payload.SourcePhaseID),
	}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnRunning(ctx context.Context, evt event.Event, payload session.AITurnRunningPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn.Status = session.AITurnStatusRunning
	current.AITurn.TurnToken = strings.TrimSpace(payload.TurnToken)
	current.AITurn.LastError = ""
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnFailed(ctx context.Context, evt event.Event, payload session.AITurnFailedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn.Status = session.AITurnStatusFailed
	current.AITurn.TurnToken = strings.TrimSpace(payload.TurnToken)
	current.AITurn.LastError = strings.TrimSpace(payload.LastError)
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionAITurnCleared(ctx context.Context, evt event.Event, _ session.AITurnClearedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.AITurn = storage.SessionAITurn{Status: session.AITurnStatusIdle}
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhaseStarted(ctx context.Context, evt event.Event, payload scene.PlayerPhaseStartedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	current.SessionID = strings.TrimSpace(evt.SessionID.String())
	current.PhaseOpen = true
	current.PhaseID = strings.TrimSpace(payload.PhaseID)
	current.PhaseStatus = scene.PlayerPhaseStatusPlayers
	current.FrameText = strings.TrimSpace(payload.FrameText)
	current.ActingCharacterIDs = characterIDsToStrings(payload.ActingCharacterIDs)
	current.ActingParticipantIDs = participantIDsToStrings(payload.ActingParticipantIDs)
	current.Slots = newScenePlayerSlots(current.ActingParticipantIDs)
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhasePosted(ctx context.Context, evt event.Event, payload scene.PlayerPhasePostedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	participantID := strings.TrimSpace(payload.ParticipantID.String())
	slot := findOrCreateScenePlayerSlot(current.Slots, participantID)
	slot.SummaryText = strings.TrimSpace(payload.SummaryText)
	slot.CharacterIDs = characterIDsToStrings(payload.CharacterIDs)
	slot.UpdatedAt = updatedAt
	slot.Yielded = false
	slot.ReviewStatus = scene.PlayerPhaseSlotReviewStatusOpen
	slot.ReviewReason = ""
	slot.ReviewCharacterIDs = []string{}
	current.Slots = upsertScenePlayerSlot(current.Slots, slot)
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhaseYielded(ctx context.Context, evt event.Event, payload scene.PlayerPhaseYieldedPayload) error {
	return a.applySceneYieldMutation(ctx, evt, strings.TrimSpace(payload.SceneID.String()), strings.TrimSpace(payload.ParticipantID.String()), true)
}

func (a Applier) applyScenePlayerPhaseReviewStarted(ctx context.Context, evt event.Event, payload scene.PlayerPhaseReviewStartedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	current.PhaseStatus = scene.PlayerPhaseStatusGMReview
	for i := range current.Slots {
		current.Slots[i].ReviewStatus = scene.PlayerPhaseSlotReviewStatusUnderReview
		current.Slots[i].ReviewReason = ""
		current.Slots[i].ReviewCharacterIDs = []string{}
	}
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhaseUnyielded(ctx context.Context, evt event.Event, payload scene.PlayerPhaseUnyieldedPayload) error {
	return a.applySceneYieldMutation(ctx, evt, strings.TrimSpace(payload.SceneID.String()), strings.TrimSpace(payload.ParticipantID.String()), false)
}

func (a Applier) applyScenePlayerPhaseRevisionsRequested(ctx context.Context, evt event.Event, payload scene.PlayerPhaseRevisionsRequestedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	current.PhaseStatus = scene.PlayerPhaseStatusPlayers
	targeted := make(map[string]scene.PlayerPhaseRevisionRequest, len(payload.Revisions))
	for _, revision := range payload.Revisions {
		targeted[strings.TrimSpace(revision.ParticipantID.String())] = revision
	}
	for i := range current.Slots {
		participantID := current.Slots[i].ParticipantID
		if revision, ok := targeted[participantID]; ok {
			current.Slots[i].Yielded = false
			current.Slots[i].ReviewStatus = scene.PlayerPhaseSlotReviewStatusChangesRequested
			current.Slots[i].ReviewReason = strings.TrimSpace(revision.Reason)
			current.Slots[i].ReviewCharacterIDs = characterIDsToStrings(revision.CharacterIDs)
			continue
		}
		current.Slots[i].ReviewStatus = scene.PlayerPhaseSlotReviewStatusAccepted
		current.Slots[i].ReviewReason = ""
		current.Slots[i].ReviewCharacterIDs = []string{}
	}
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhaseAccepted(ctx context.Context, evt event.Event, payload scene.PlayerPhaseAcceptedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	for i := range current.Slots {
		current.Slots[i].ReviewStatus = scene.PlayerPhaseSlotReviewStatusAccepted
		current.Slots[i].ReviewReason = ""
		current.Slots[i].ReviewCharacterIDs = []string{}
	}
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applyScenePlayerPhaseEnded(ctx context.Context, evt event.Event, payload scene.PlayerPhaseEndedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	current.PhaseOpen = false
	current.PhaseID = ""
	current.PhaseStatus = ""
	current.FrameText = ""
	current.ActingCharacterIDs = []string{}
	current.ActingParticipantIDs = []string{}
	current.Slots = []storage.ScenePlayerSlot{}
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

func (a Applier) applySceneYieldMutation(ctx context.Context, evt event.Event, payloadSceneID, participantID string, yielded bool) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sceneID, err := resolveSceneID(payloadSceneID, evt.SceneID)
	if err != nil {
		return err
	}
	current, err := loadSceneInteraction(ctx, a.SceneInteraction, string(evt.CampaignID), sceneID, evt.SessionID.String())
	if err != nil {
		return err
	}
	participantID = strings.TrimSpace(participantID)
	slot := findOrCreateScenePlayerSlot(current.Slots, participantID)
	slot.Yielded = yielded
	if !yielded {
		slot.ReviewStatus = scene.PlayerPhaseSlotReviewStatusOpen
		slot.ReviewReason = ""
		slot.ReviewCharacterIDs = []string{}
	}
	current.Slots = upsertScenePlayerSlot(current.Slots, slot)
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
}

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
	filtered := values[:0]
	for _, value := range values {
		if value != target {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
