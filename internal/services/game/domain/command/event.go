package command

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// NewEvent builds an event.Event by copying the shared envelope fields from a
// command. Callers supply the event-specific type, entity addressing, payload,
// and timestamp. This eliminates per-decider boilerplate and ensures that new
// envelope fields are automatically forwarded.
func NewEvent(cmd Command, eventType event.Type, entityType, entityID string, payloadJSON []byte, now time.Time) event.Event {
	return event.Event{
		CampaignID:    cmd.CampaignID,
		Type:          eventType,
		Timestamp:     now,
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    entityType,
		EntityID:      entityID,
		SystemID:      cmd.SystemID,
		SystemVersion: cmd.SystemVersion,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		PayloadJSON:   payloadJSON,
	}
}
