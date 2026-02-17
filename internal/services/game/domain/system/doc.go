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
package system
