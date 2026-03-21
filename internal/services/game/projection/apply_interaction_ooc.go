package projection

import (
	"context"
	"slices"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionOOCOpened(ctx context.Context, evt event.Event, payload session.OOCOpenedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.OOCPaused = true
	current.OOCRequestedByParticipantID = strings.TrimSpace(payload.RequestedByParticipantID.String())
	current.OOCReason = strings.TrimSpace(payload.Reason)
	current.OOCInterruptedSceneID = strings.TrimSpace(payload.InterruptedSceneID.String())
	current.OOCInterruptedPhaseID = strings.TrimSpace(payload.InterruptedPhaseID)
	current.OOCInterruptedPhaseStatus = strings.TrimSpace(payload.InterruptedPhaseStatus)
	current.OOCResolutionPending = false
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

func (a Applier) applySessionOOCClosed(ctx context.Context, evt event.Event, _ session.OOCClosedPayload) error {
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
	current.OOCResolutionPending = current.OOCInterruptedSceneID != "" && current.OOCInterruptedPhaseID != ""
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}

func (a Applier) applySessionOOCResolved(ctx context.Context, evt event.Event, _ session.OOCResolvedPayload) error {
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	current, err := loadSessionInteraction(ctx, a.SessionInteraction, string(evt.CampaignID), evt.SessionID.String())
	if err != nil {
		return err
	}
	current.OOCRequestedByParticipantID = ""
	current.OOCReason = ""
	current.OOCInterruptedSceneID = ""
	current.OOCInterruptedPhaseID = ""
	current.OOCInterruptedPhaseStatus = ""
	current.OOCResolutionPending = false
	current.UpdatedAt = updatedAt
	return a.SessionInteraction.PutSessionInteraction(ctx, current)
}
