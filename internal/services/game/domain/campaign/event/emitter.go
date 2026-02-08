package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Store defines the interface for persisting events.
type Store interface {
	AppendEvent(ctx context.Context, evt Event) (Event, error)
}

// Emitter provides event emission capabilities for state mutations.
type Emitter struct {
	store Store
	now   func() time.Time
}

// NewEmitter creates a new event emitter.
func NewEmitter(store Store) *Emitter {
	return &Emitter{
		store: store,
		now:   time.Now,
	}
}

// EmitInput describes the input for emitting an event.
type EmitInput struct {
	CampaignID   string
	Type         Type
	SessionID    string
	RequestID    string
	InvocationID string
	ActorType    ActorType
	ActorID      string
	EntityType   string
	EntityID     string
	Payload      any
}

// Emit appends an event to the unified event journal.
func (e *Emitter) Emit(ctx context.Context, input EmitInput) (Event, error) {
	if e.store == nil {
		return Event{}, fmt.Errorf("event store is not configured")
	}

	payloadJSON, err := json.Marshal(input.Payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal event payload: %w", err)
	}

	evt := Event{
		CampaignID:   input.CampaignID,
		Timestamp:    e.now().UTC(),
		Type:         input.Type,
		SessionID:    input.SessionID,
		RequestID:    input.RequestID,
		InvocationID: input.InvocationID,
		ActorType:    input.ActorType,
		ActorID:      input.ActorID,
		EntityType:   input.EntityType,
		EntityID:     input.EntityID,
		PayloadJSON:  payloadJSON,
	}

	return e.store.AppendEvent(ctx, evt)
}

// EmitCampaignCreated emits a campaign.created event.
func (e *Emitter) EmitCampaignCreated(ctx context.Context, campaignID string, payload CampaignCreatedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeCampaignCreated,
		ActorType:  ActorTypeSystem,
		EntityType: "campaign",
		EntityID:   campaignID,
		Payload:    payload,
	})
}

// EmitCampaignForked emits a campaign.forked event.
func (e *Emitter) EmitCampaignForked(ctx context.Context, campaignID string, payload CampaignForkedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeCampaignForked,
		ActorType:  ActorTypeSystem,
		EntityType: "campaign",
		EntityID:   campaignID,
		Payload:    payload,
	})
}

// EmitCampaignStatusChanged emits a campaign.status_changed event.
func (e *Emitter) EmitCampaignStatusChanged(ctx context.Context, campaignID, actorID string, payload CampaignStatusChangedPayload) (Event, error) {
	actorType := ActorTypeSystem
	if actorID != "" {
		actorType = ActorTypeParticipant
	}
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeCampaignStatusChanged,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "campaign",
		EntityID:   campaignID,
		Payload:    payload,
	})
}

// EmitParticipantJoined emits a participant.joined event.
func (e *Emitter) EmitParticipantJoined(ctx context.Context, campaignID string, payload ParticipantJoinedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeParticipantJoined,
		ActorType:  ActorTypeSystem,
		EntityType: "participant",
		EntityID:   payload.ParticipantID,
		Payload:    payload,
	})
}

// EmitCharacterCreated emits a character.created event.
func (e *Emitter) EmitCharacterCreated(ctx context.Context, campaignID string, payload CharacterCreatedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeCharacterCreated,
		ActorType:  ActorTypeSystem,
		EntityType: "character",
		EntityID:   payload.CharacterID,
		Payload:    payload,
	})
}

// EmitProfileUpdated emits a character.profile_updated event.
func (e *Emitter) EmitProfileUpdated(ctx context.Context, campaignID, actorID string, payload ProfileUpdatedPayload) (Event, error) {
	actorType := ActorTypeSystem
	if actorID != "" {
		actorType = ActorTypeGM
	}
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeProfileUpdated,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "character",
		EntityID:   payload.CharacterID,
		Payload:    payload,
	})
}

// EmitControllerAssigned emits a character.controller_assigned event.
func (e *Emitter) EmitControllerAssigned(ctx context.Context, campaignID, actorID string, payload ControllerAssignedPayload) (Event, error) {
	actorType := ActorTypeSystem
	if actorID != "" {
		actorType = ActorTypeGM
	}
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeControllerAssigned,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "character",
		EntityID:   payload.CharacterID,
		Payload:    payload,
	})
}

// EmitCharacterStateChanged emits a snapshot character state changed event.
func (e *Emitter) EmitCharacterStateChanged(ctx context.Context, campaignID, sessionID, actorID string, payload CharacterStateChangedPayload) (Event, error) {
	actorType := ActorTypeSystem
	if actorID != "" {
		actorType = ActorTypeGM
	}
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeCharacterStateChanged,
		SessionID:  sessionID,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "character",
		EntityID:   payload.CharacterID,
		Payload:    payload,
	})
}

// EmitGMFearChanged emits a snapshot GM fear changed event.
func (e *Emitter) EmitGMFearChanged(ctx context.Context, campaignID, sessionID, actorID string, payload GMFearChangedPayload) (Event, error) {
	actorType := ActorTypeSystem
	if actorID != "" {
		actorType = ActorTypeGM
	}
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeGMFearChanged,
		SessionID:  sessionID,
		ActorType:  actorType,
		ActorID:    actorID,
		EntityType: "snapshot",
		EntityID:   campaignID,
		Payload:    payload,
	})
}

// EmitSessionStarted emits a session.started event.
func (e *Emitter) EmitSessionStarted(ctx context.Context, campaignID string, payload SessionStartedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeSessionStarted,
		SessionID:  payload.SessionID,
		ActorType:  ActorTypeSystem,
		EntityType: "session",
		EntityID:   payload.SessionID,
		Payload:    payload,
	})
}

// EmitSessionEnded emits a session.ended event.
func (e *Emitter) EmitSessionEnded(ctx context.Context, campaignID string, payload SessionEndedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeSessionEnded,
		SessionID:  payload.SessionID,
		ActorType:  ActorTypeSystem,
		EntityType: "session",
		EntityID:   payload.SessionID,
		Payload:    payload,
	})
}

// EmitRollResolved emits an action.roll_resolved event.
func (e *Emitter) EmitRollResolved(ctx context.Context, campaignID, sessionID string, payload RollResolvedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeRollResolved,
		SessionID:  sessionID,
		ActorType:  ActorTypeSystem,
		EntityType: "roll",
		EntityID:   payload.RequestID,
		Payload:    payload,
	})
}

// EmitOutcomeApplied emits an action.outcome_applied event.
func (e *Emitter) EmitOutcomeApplied(ctx context.Context, campaignID, sessionID string, payload OutcomeAppliedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeOutcomeApplied,
		SessionID:  sessionID,
		ActorType:  ActorTypeSystem,
		EntityType: "outcome",
		EntityID:   payload.RequestID,
		Payload:    payload,
	})
}

// EmitOutcomeRejected emits an action.outcome_rejected event.
func (e *Emitter) EmitOutcomeRejected(ctx context.Context, campaignID, sessionID string, payload OutcomeRejectedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeOutcomeRejected,
		SessionID:  sessionID,
		ActorType:  ActorTypeSystem,
		EntityType: "outcome",
		EntityID:   payload.RequestID,
		Payload:    payload,
	})
}

// EmitNoteAdded emits an action.note_added event.
func (e *Emitter) EmitNoteAdded(ctx context.Context, campaignID, sessionID, actorID string, payload NoteAddedPayload) (Event, error) {
	return e.Emit(ctx, EmitInput{
		CampaignID: campaignID,
		Type:       TypeNoteAdded,
		SessionID:  sessionID,
		ActorType:  ActorTypeParticipant,
		ActorID:    actorID,
		EntityType: "note",
		Payload:    payload,
	})
}
