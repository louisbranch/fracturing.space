// Package event provides unified event types for campaign event sourcing.
//
// All state changes in the system are recorded as immutable events in a
// per-campaign event log. This enables forking, auditing, and replay.
//
// Events are organized by domain:
//   - campaign.*: campaign lifecycle (created, forked, status changes)
//   - participant.*: participant membership changes
//   - character.*: character definitions and profile updates
//   - snapshot.*: snapshot state changes (character state, GM fear)
//   - session.*: session lifecycle
//   - action.*: gameplay actions (rolls, outcomes, notes). System-owned action
//     events live in system packages and are tagged with system metadata.
//
// Each event type has a corresponding payload struct serialized as JSON.
package event
