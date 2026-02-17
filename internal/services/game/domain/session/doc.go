// Package session models the session aggregate.
//
// Sessions track live game tempo inside an active campaign:
// start/end lifecycle, spotlight control, and temporary adjudication gates that
// pause certain command classes.
//
// For onboarding, this package is the source of truth for what is considered
// "currently in session" versus "currently blocked by GM pause."
//
// The package holds:
//   - command deciders that translate session commands into events,
//   - fold logic for replaying session history,
//   - and state constraints used by the command gate.
package session
