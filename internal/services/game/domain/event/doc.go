// Package event defines the canonical event envelope and event-type registry
// used by the game domain write path.
//
// Events are immutable facts emitted by accepted decisions. Event registry
// validation enforces ownership rules (core vs system), actor metadata, payload
// validity, and append-time invariants before storage assigns sequence and
// integrity fields.
package event
