package projection

import (
	"context"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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

func (a Applier) applySceneGMOutputCommitted(ctx context.Context, evt event.Event, payload scene.GMOutputCommittedPayload) error {
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
	current.GMOutputText = strings.TrimSpace(payload.Text)
	current.GMOutputParticipantID = strings.TrimSpace(payload.ParticipantID.String())
	current.GMOutputUpdatedAt = timePtr(updatedAt)
	current.UpdatedAt = updatedAt
	return a.SceneInteraction.PutSceneInteraction(ctx, current)
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

func timePtr(value time.Time) *time.Time {
	result := value
	return &result
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
