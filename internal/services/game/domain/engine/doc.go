// Package engine wires command validation, gate checks, decision routing, event
// append, and replay-backed state loading for domain command execution.
//
// This package is the runtime seam between immutable domain contracts and
// transport handlers: it validates intent, applies domain policy, persists events,
// and returns a replayable decision/result for downstream handlers.
package engine
