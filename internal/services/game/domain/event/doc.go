// Package event defines the canonical event envelope and event-type registry used by
// the game domain write path.
//
// Events are immutable business facts emitted by accepted decisions. The registry
// enforces ownership boundaries (core vs system), actor metadata, and payload
// validity before persistence assigns sequence and integrity fields.
//
// A stable event contract is the foundation for replay, projection correctness,
// and cross-service consumers that depend on the same semantic names.
package event
