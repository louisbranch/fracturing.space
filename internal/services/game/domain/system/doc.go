// Package system defines the pluggable game-system layer.
//
// Core and system-owned types share one domain envelope, but system modules are
// treated as independent plug-ins that contribute:
//   - command/event registration,
//   - domain decision logic for system commands,
//   - projection logic for system events,
//   - and state factories for per-system snapshots.
//
// This boundary is what lets new game systems be added without changing the
// core campaign/session/participant command flow.
//
// Note: this package (domain/system) handles module registration and command/event
// routing for the write model. The sibling package domain/systems provides the
// projection-side adapter registry and the API metadata bridge (proto enum mapping).
// The two packages collaborate but own different concerns.
package system
