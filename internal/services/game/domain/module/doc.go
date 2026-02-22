// Package module defines the pluggable game-system layer for the write model.
//
// Each game system (e.g. Daggerheart) is registered as a Module that contributes:
//   - command/event registration,
//   - domain decision logic for system commands,
//   - projection logic for system events in the aggregate,
//   - and state factories for per-system snapshots.
//
// This boundary is what lets new game systems be added without changing the
// core campaign/session/participant command flow.
//
// Compare with the sibling package domain/bridge, which provides the
// projection-side adapter registry and the API metadata bridge (proto enum
// mapping). The two packages collaborate but own different concerns:
//
//   - module.Registry  — write-path module routing (commands → decider, events → projector)
//   - bridge.AdapterRegistry — projection-side adapters for system-specific read models
package module
