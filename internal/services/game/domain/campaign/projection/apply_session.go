package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applySessionStarted(ctx context.Context, evt event.Event) error {
	if a.Session == nil {
		return fmt.Errorf("session store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.SessionStartedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.started payload: %w", err)
	}
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	startedAt := ensureTimestamp(evt.Timestamp)
	return a.Session.PutSession(ctx, session.Session{
		ID:         sessionID,
		CampaignID: evt.CampaignID,
		Name:       strings.TrimSpace(payload.SessionName),
		Status:     session.SessionStatusActive,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
	})
}

func (a Applier) applySessionEnded(ctx context.Context, evt event.Event) error {
	if a.Session == nil {
		return fmt.Errorf("session store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.SessionEndedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.ended payload: %w", err)
	}
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(evt.EntityID)
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	_, _, err := a.Session.EndSession(ctx, evt.CampaignID, sessionID, ensureTimestamp(evt.Timestamp))
	return err
}

func (a Applier) applySessionGateOpened(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload event.SessionGateOpenedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.gate_opened payload: %w", err)
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
	createdAt := ensureTimestamp(evt.Timestamp)
	return a.SessionGate.PutSessionGate(ctx, storage.SessionGate{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		GateID:             gateID,
		GateType:           gateType,
		Status:             string(session.GateStatusOpen),
		Reason:             reason,
		CreatedAt:          createdAt,
		CreatedByActorType: string(evt.ActorType),
		CreatedByActorID:   evt.ActorID,
		MetadataJSON:       metadataJSON,
	})
}

func (a Applier) applySessionGateResolved(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload event.SessionGateResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.gate_resolved payload: %w", err)
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
	resolvedAt := ensureTimestamp(evt.Timestamp)
	gate.Status = string(session.GateStatusResolved)
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionGateAbandoned(ctx context.Context, evt event.Event) error {
	if a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload event.SessionGateAbandonedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.gate_abandoned payload: %w", err)
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
	resolvedAt := ensureTimestamp(evt.Timestamp)
	gate.Status = string(session.GateStatusAbandoned)
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = string(evt.ActorType)
	gate.ResolvedByActorID = evt.ActorID
	gate.ResolutionJSON = resolutionJSON
	return a.SessionGate.PutSessionGate(ctx, gate)
}

func (a Applier) applySessionSpotlightSet(ctx context.Context, evt event.Event) error {
	if a.SessionSpotlight == nil {
		return fmt.Errorf("session spotlight store is not configured")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	var payload event.SessionSpotlightSetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode session.spotlight_set payload: %w", err)
	}
	spotlightType, err := session.NormalizeSpotlightType(payload.SpotlightType)
	if err != nil {
		return err
	}
	if err := session.ValidateSpotlightTarget(spotlightType, payload.CharacterID); err != nil {
		return err
	}

	return a.SessionSpotlight.PutSessionSpotlight(ctx, storage.SessionSpotlight{
		CampaignID:         evt.CampaignID,
		SessionID:          evt.SessionID,
		SpotlightType:      string(spotlightType),
		CharacterID:        strings.TrimSpace(payload.CharacterID),
		UpdatedAt:          ensureTimestamp(evt.Timestamp),
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySessionSpotlightCleared(ctx context.Context, evt event.Event) error {
	if a.SessionSpotlight == nil {
		return fmt.Errorf("session spotlight store is not configured")
	}
	if strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	return a.SessionSpotlight.ClearSessionSpotlight(ctx, evt.CampaignID, evt.SessionID)
}
