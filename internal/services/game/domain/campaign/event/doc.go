// Package event provides unified event types for campaign event sourcing.
//
// All state changes in the system are recorded as immutable events in a
// per-campaign event log. This enables:
//
//   - Campaign forking at any event point
//   - Full audit trail of all changes
//   - State reconstruction via event replay
//   - Snapshots captured at event sequences for replay acceleration
//
// Event Types:
//
// Events are organized by domain:
//
//   - campaign.*: Campaign lifecycle (created, forked, status changes)
//   - participant.*: Participant management (joined, left, updated)
//   - character.*: Character definitions and profiles
//   - snapshot.*: System state change events (legacy)
//   - action.*: Gameplay actions (rolls, outcomes)
//   - story.*: Narrative changes (notes, canon, scene progression)
//
// Story Event Example:
//
// Story events capture narrative updates in the same journal as other gameplay
// changes. They are not yet surfaced through user-facing tooling, but they
// follow the same append/validation pipeline as any other event.
//
//	Type:       "story.note_added"
//	ActorType:  "system"
//	EntityType: "session"
//	EntityId:   "session_123"
//	Payload:
//	  {"content":"The lantern sputters as the door seals behind you."}
//
// Event Payloads:
//
// Each event type has a corresponding payload struct that captures the
// event-specific data. Payloads are serialized as JSON in the database.
package event
