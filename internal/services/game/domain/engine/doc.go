// Package engine wires command validation, gate checks, lifecycle policy,
// decision routing, event append, and replay-backed state loading for domain
// command execution.
//
// This package is the runtime seam between immutable domain contracts and
// transport handlers: it validates intent, applies domain policy, persists events,
// and returns a replayable decision/result for downstream handlers.
//
// Session start remains the one intentional cross-aggregate exception inside
// the core write path. CoreDecider stays thin by delegating core-owned routing
// to a dedicated core router and system-owned envelopes to a system dispatcher,
// while the readiness-owned workflow seam keeps campaign activation logic out of
// the engine entrypoint.
package engine
