package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionStarted(ctx context.Context, evt event.Event, payload session.StartPayload) error {
	sessionID := strings.TrimSpace(payload.SessionID.String())
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	startedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	if err := a.Session.PutSession(ctx, storage.SessionRecord{
		ID:         sessionID,
		CampaignID: string(evt.CampaignID),
		Name:       strings.TrimSpace(payload.SessionName),
		Status:     session.StatusActive,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
	}); err != nil {
		return err
	}
	if a.SessionInteraction == nil {
		return nil
	}
	return a.SessionInteraction.PutSessionInteraction(ctx, storage.SessionInteraction{
		CampaignID:                  string(evt.CampaignID),
		SessionID:                   sessionID,
		ReadyToResumeParticipantIDs: []string{},
		UpdatedAt:                   startedAt,
	})
}

func (a Applier) applySessionEnded(ctx context.Context, evt event.Event, payload session.EndPayload) error {
	sessionID := strings.TrimSpace(payload.SessionID.String())
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	endedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	_, _, err = a.Session.EndSession(ctx, string(evt.CampaignID), sessionID, endedAt)
	if err != nil {
		return err
	}
	if a.SessionInteraction == nil {
		return nil
	}
	return a.SessionInteraction.PutSessionInteraction(ctx, storage.SessionInteraction{
		CampaignID:                  string(evt.CampaignID),
		SessionID:                   sessionID,
		ReadyToResumeParticipantIDs: []string{},
		UpdatedAt:                   endedAt,
	})
}

func (a Applier) applySessionGateOpened(ctx context.Context, evt event.Event, payload session.GateOpenedPayload) error {
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gateType, err := session.NormalizeGateType(payload.GateType)
	if err != nil {
		return err
	}
	reason := session.NormalizeGateReason(payload.Reason)
	metadata, err := session.NormalizeGateWorkflowMetadata(gateType, payload.Metadata)
	if err != nil {
		return fmt.Errorf("normalize gate metadata: %w", err)
	}
	progress, err := session.BuildInitialGateProgressState(gateType, metadata)
	if err != nil {
		return fmt.Errorf("build gate progress: %w", err)
	}
	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SessionGate.PutSessionGate(ctx, storage.SessionGate{
		CampaignID:         string(evt.CampaignID),
		SessionID:          evt.SessionID.String(),
		GateID:             gateID,
		GateType:           gateType,
		Status:             session.GateStatusOpen,
		Reason:             reason,
		CreatedAt:          createdAt,
		CreatedByActorType: string(evt.ActorType),
		CreatedByActorID:   evt.ActorID,
		Metadata:           metadata,
		Progress:           progress,
	})
}

func (a Applier) applySessionGateResponseRecorded(ctx context.Context, evt event.Event, payload session.GateResponseRecordedPayload) error {
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, string(evt.CampaignID), evt.SessionID.String(), gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	recordedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	progress, err := session.RecordGateResponseProgressState(
		gate.GateType,
		gate.Metadata,
		gate.Progress,
		payload,
		recordedAt,
		string(evt.ActorType),
		evt.ActorID,
	)
	if err != nil {
		return fmt.Errorf("record gate response progress: %w", err)
	}
	gate.Progress = progress
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateResolved(ctx context.Context, evt event.Event, payload session.GateResolvedPayload) error {
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, string(evt.CampaignID), evt.SessionID.String(), gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolution, err := session.BuildGateResolutionMap(payload.Decision, payload.Resolution)
	if err != nil {
		return fmt.Errorf("build gate resolution: %w", err)
	}
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.Resolution = resolution
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateAbandoned(ctx context.Context, evt event.Event, payload session.GateAbandonedPayload) error {
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, string(evt.CampaignID), evt.SessionID.String(), gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolution, err := session.BuildGateResolutionMap("abandoned", map[string]any{"reason": session.NormalizeGateReason(payload.Reason)})
	if err != nil {
		return fmt.Errorf("build gate resolution: %w", err)
	}
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusAbandoned
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.Resolution = resolution
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionSpotlightSet(ctx context.Context, evt event.Event, payload session.SpotlightSetPayload) error {
	spotlightType, err := session.NormalizeSpotlightType(payload.SpotlightType)
	if err != nil {
		return err
	}
	if err := session.ValidateSpotlightTarget(spotlightType, payload.CharacterID.String()); err != nil {
		return err
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SessionSpotlight.PutSessionSpotlight(ctx, storage.SessionSpotlight{
		CampaignID:         string(evt.CampaignID),
		SessionID:          evt.SessionID.String(),
		SpotlightType:      spotlightType,
		CharacterID:        strings.TrimSpace(payload.CharacterID.String()),
		UpdatedAt:          updatedAt,
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySessionSpotlightCleared(ctx context.Context, evt event.Event) error {
	return a.SessionSpotlight.ClearSessionSpotlight(ctx, string(evt.CampaignID), evt.SessionID.String())
}
