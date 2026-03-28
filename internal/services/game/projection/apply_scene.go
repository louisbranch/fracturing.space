package projection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// resolveSceneID extracts the scene ID from the payload with a fallback to the
// event envelope's SceneID. This mirrors the session pattern of checking the
// payload first and falling back to EntityID.
func resolveSceneID(payloadSceneID string, envelopeSceneID ids.SceneID) (string, error) {
	sceneID := strings.TrimSpace(payloadSceneID)
	if sceneID == "" {
		sceneID = strings.TrimSpace(envelopeSceneID.String())
	}
	if sceneID == "" {
		return "", fmt.Errorf("scene id is required")
	}
	return sceneID, nil
}

func (a Applier) applySceneCreated(ctx context.Context, evt event.Event, payload scene.CreatePayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	sessionID := strings.TrimSpace(evt.SessionID.String())
	if sessionID == "" {
		return fmt.Errorf("session id is required for scene creation")
	}

	if err := a.Scene.PutScene(ctx, storage.SceneRecord{
		CampaignID:  string(evt.CampaignID),
		SceneID:     sceneID,
		SessionID:   sessionID,
		Name:        strings.TrimSpace(payload.Name),
		Description: strings.TrimSpace(payload.Description),
		Open:        true,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}); err != nil {
		return err
	}
	if err := a.SceneInteraction.PutSceneInteraction(ctx, storage.SceneInteraction{
		CampaignID:           string(evt.CampaignID),
		SceneID:              sceneID,
		SessionID:            sessionID,
		ActingCharacterIDs:   []string{},
		ActingParticipantIDs: []string{},
		Slots:                []storage.ScenePlayerSlot{},
		UpdatedAt:            createdAt,
	}); err != nil {
		return err
	}

	// Add initial characters.
	for _, charID := range payload.CharacterIDs {
		charIDStr := strings.TrimSpace(charID.String())
		if charIDStr == "" {
			continue
		}
		if err := a.SceneCharacter.PutSceneCharacter(ctx, storage.SceneCharacterRecord{
			CampaignID:  string(evt.CampaignID),
			SceneID:     sceneID,
			CharacterID: charIDStr,
			AddedAt:     createdAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a Applier) applySceneUpdated(ctx context.Context, evt event.Event, payload scene.UpdatePayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}

	existing, err := a.Scene.GetScene(ctx, string(evt.CampaignID), sceneID)
	if err != nil {
		return fmt.Errorf("get scene for update: %w", err)
	}

	if name := strings.TrimSpace(payload.Name); name != "" {
		existing.Name = name
	}
	if desc := strings.TrimSpace(payload.Description); desc != "" {
		existing.Description = desc
	}
	existing.UpdatedAt = updatedAt

	return a.Scene.PutScene(ctx, existing)
}

func (a Applier) applySceneEnded(ctx context.Context, evt event.Event, payload scene.EndPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	endedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}

	if err := a.Scene.EndScene(ctx, string(evt.CampaignID), sceneID, endedAt); err != nil {
		return err
	}
	sceneState, err := a.SceneInteraction.GetSceneInteraction(ctx, string(evt.CampaignID), sceneID)
	if err != nil {
		return fmt.Errorf("get scene interaction on end: %w", err)
	}
	sceneState.PhaseOpen = false
	sceneState.PhaseID = ""
	sceneState.PhaseStatus = ""
	sceneState.ActingCharacterIDs = []string{}
	sceneState.ActingParticipantIDs = []string{}
	sceneState.Slots = []storage.ScenePlayerSlot{}
	sceneState.UpdatedAt = endedAt
	if err := a.SceneInteraction.PutSceneInteraction(ctx, sceneState); err != nil {
		return fmt.Errorf("put scene interaction on end: %w", err)
	}
	// Clear spotlight when scene ends. Suppress ErrNotFound (no spotlight was
	// set), but propagate real storage errors to avoid inconsistent state.
	if err := a.SceneSpotlight.ClearSceneSpotlight(ctx, string(evt.CampaignID), sceneID); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("clear scene spotlight on end: %w", err)
	}
	return nil
}

func (a Applier) applySceneCharacterAdded(ctx context.Context, evt event.Event, payload scene.CharacterAddedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	addedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	charID := strings.TrimSpace(payload.CharacterID.String())
	if charID == "" {
		return fmt.Errorf("character id is required")
	}
	return a.SceneCharacter.PutSceneCharacter(ctx, storage.SceneCharacterRecord{
		CampaignID:  string(evt.CampaignID),
		SceneID:     sceneID,
		CharacterID: charID,
		AddedAt:     addedAt,
	})
}

func (a Applier) applySceneCharacterRemoved(ctx context.Context, evt event.Event, payload scene.CharacterRemovedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	charID := strings.TrimSpace(payload.CharacterID.String())
	if charID == "" {
		return fmt.Errorf("character id is required")
	}
	return a.SceneCharacter.DeleteSceneCharacter(ctx, string(evt.CampaignID), sceneID, charID)
}

func (a Applier) applySceneGateOpened(ctx context.Context, evt event.Event, payload scene.GateOpenedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gateType := strings.TrimSpace(payload.GateType)
	if gateType == "" {
		return fmt.Errorf("gate type is required")
	}
	metadataJSON, err := marshalOptionalMap(payload.Metadata)
	if err != nil {
		return fmt.Errorf("encode gate metadata: %w", err)
	}
	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SceneGate.PutSceneGate(ctx, storage.SceneGate{
		CampaignID:         string(evt.CampaignID),
		SceneID:            sceneID,
		GateID:             gateID,
		GateType:           gateType,
		Status:             session.GateStatusOpen,
		Reason:             strings.TrimSpace(payload.Reason),
		CreatedAt:          createdAt,
		CreatedByActorType: string(evt.ActorType),
		CreatedByActorID:   evt.ActorID,
		MetadataJSON:       metadataJSON,
	})
}

func (a Applier) applySceneGateResolved(ctx context.Context, evt event.Event, payload scene.GateResolvedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SceneGate.GetSceneGate(ctx, string(evt.CampaignID), sceneID, gateID)
	if err != nil {
		return fmt.Errorf("get scene gate: %w", err)
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
	return a.SceneGate.PutSceneGate(ctx, gate)
}

func (a Applier) applySceneGateAbandoned(ctx context.Context, evt event.Event, payload scene.GateAbandonedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	gateID := strings.TrimSpace(payload.GateID.String())
	if gateID == "" {
		gateID = strings.TrimSpace(evt.EntityID)
	}
	if gateID == "" {
		return fmt.Errorf("gate id is required")
	}
	gate, err := a.SceneGate.GetSceneGate(ctx, string(evt.CampaignID), sceneID, gateID)
	if err != nil {
		return fmt.Errorf("get scene gate: %w", err)
	}
	resolutionJSON, err := marshalResolutionPayload("abandoned", map[string]any{"reason": strings.TrimSpace(payload.Reason)})
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
	return a.SceneGate.PutSceneGate(ctx, gate)
}

func (a Applier) applySceneSpotlightSet(ctx context.Context, evt event.Event, payload scene.SpotlightSetPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	spotlightType, err := scene.NormalizeSpotlightType(string(payload.SpotlightType))
	if err != nil {
		return err
	}
	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.SceneSpotlight.PutSceneSpotlight(ctx, storage.SceneSpotlight{
		CampaignID:         string(evt.CampaignID),
		SceneID:            sceneID,
		SpotlightType:      spotlightType,
		CharacterID:        strings.TrimSpace(payload.CharacterID.String()),
		UpdatedAt:          updatedAt,
		UpdatedByActorType: string(evt.ActorType),
		UpdatedByActorID:   evt.ActorID,
	})
}

func (a Applier) applySceneSpotlightCleared(ctx context.Context, evt event.Event, payload scene.SpotlightClearedPayload) error {
	sceneID, err := resolveSceneID(payload.SceneID.String(), evt.SceneID)
	if err != nil {
		return err
	}
	return a.SceneSpotlight.ClearSceneSpotlight(ctx, string(evt.CampaignID), sceneID)
}
