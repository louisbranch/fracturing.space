// Package action captures gameplay action signals that are usually system-owned.
//
// Actions are intentionally light-weight:
// - this aggregate does not own long-lived per-entity state,
// - it validates action intent,
// - and emits immutable action events for system modules and projections to interpret.
//
// This keeps the core write path uniform while allowing systems (like Daggerheart)
// to evolve result semantics independently without changing campaign/session
// aggregate structure.
package action
