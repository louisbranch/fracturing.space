// Package scene models the scene aggregate.
//
// Scenes are narrative sub-session scopes that enable split-party play,
// parallel timelines, information isolation, and per-scene gate/spotlight
// mechanics. A session may have zero or more active scenes; each scene
// tracks its own character roster, gate state, and spotlight independently.
//
// The package holds:
//   - workflow-local command deciders that translate scene commands into events,
//   - fold logic for replaying scene history,
//   - and state constraints used by the scene-scoped gate.
//
// Scene command handling is intentionally split by capability inside the
// package so lifecycle, character, gate, and spotlight workflows stay
// navigable without changing the aggregate boundary.
package scene
