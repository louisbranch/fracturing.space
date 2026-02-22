package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionStarted(ctx context.Context, evt event.Event) error {
	var payload session.StartPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.started"); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(payload.SessionID)
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
	return a.Session.PutSession(ctx, storage.SessionRecord{
		ID:         sessionID,
		CampaignID: evt.CampaignID,
		Name:       strings.TrimSpace(payload.SessionName),
		Status:     session.StatusActive,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
	})
}

func (a Applier) applySessionEnded(ctx context.Context, evt event.Event) error {
	var payload session.EndPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.ended"); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(payload.SessionID)
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
	_, _, err = a.Session.EndSession(ctx, evt.CampaignID, sessionID, endedAt)
	return err
}

func (a Applier) applySessionGateOpened(ctx context.Context, evt event.Event) error {
	var payload session.GateOpenedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_opened"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
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
	metadataJSON, err := marshalOptionalMap(payload.Metadata)
	if err != nil {
		return fmt.Errorf("encode gate metadata: %w", err)
	}
	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SessionGate.PutSessionGate(ctx, storage.SessionGate{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		GateID:             gateID,
		GateType:           gateType,
		Status:             session.GateStatusOpen,
		Reason:             reason,
		CreatedAt:          createdAt,
		CreatedByActorType: string(evt.ActorType),
		CreatedByActorID:   evt.ActorID,
		MetadataJSON:       metadataJSON,
	})
}

func (a Applier) applySessionGateResolved(ctx context.Context, evt event.Event) error {
	var payload session.GateResolvedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_resolved"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, evt.CampaignID, evt.SessionID, gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolutionJSON, err := marshalResolutionPayload(payload.Decision, payload.Resolution)
	if err != nil {
		return fmt.Errorf("encode gate resolution: %w", err)
	}
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateAbandoned(ctx context.Context, evt event.Event) error {
	var payload session.GateAbandonedPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.gate_abandoned"); err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID)
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SessionGate.GetSessionGate(ctx, evt.CampaignID, evt.SessionID, gateID)
	if err != nil {
		return fmt.Errorf("get session gate: %w", err)
	}
	resolutionJSON, err := marshalResolutionPayload("abandoned", map[string]any{"reason": session.NormalizeGateReason(payload.Reason)})
	if err != nil {
		return fmt.Errorf("encode gate resolution: %w", err)
	}
	resolvedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	gate.Status = session.GateStatusAbandoned
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionSpotlightSet(ctx context.Context, evt event.Event) error {
	var payload session.SpotlightSetPayload
	if err := decodePayload(evt.PayloadJSON, &payload, "session.spotlight_set"); err != nil {
		return err
	}
	spotlightType, err := session.NormalizeSpotlightType(payload.SpotlightType)
	if err != nil {
		return err
	}
	if err := session.ValidateSpotlightTarget(spotlightType, payload.CharacterID); err != nil {
		return err
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SessionSpotlight.PutSessionSpotlight(ctx, storage.SessionSpotlight{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		SpotlightType:      spotlightType,
		CharacterID:        strings.TrimSpace(payload.CharacterID),
		UpdatedAt:          updatedAt,
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySessionSpotlightCleared(ctx context.Context, evt event.Event) error {
	return a.SessionSpotlight.ClearSessionSpotlight(ctx, evt.CampaignID, evt.SessionID)
}
