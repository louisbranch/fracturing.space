// Package session models the session aggregate.
//
// Sessions track live game tempo inside an active campaign:
// start/end lifecycle, spotlight control, and temporary adjudication gates that
// pause certain command classes.
//
// For onboarding, this package is the source of truth for what is considered
// "currently in session" versus "currently blocked by GM pause." It is also the
// replay-owned authority for session-gate workflow validation. Projection-layer
// gate summaries may be richer for UI purposes, but they are derived from the
// rules defined here.
//
// The package holds:
//   - command deciders that translate session commands into events,
//   - fold logic for replaying session history,
//   - state constraints used by the command gate,
//   - and workflow normalization/validation for active session gates.
//
// Read-model-only gate progress belongs outside this package. If a gate summary
// exists only to support transport/UI reads, it should not become aggregate
// authority by living here.
package session
