// Package readiness evaluates campaign/session readiness invariants that gate
// session start decisions and owns the cross-aggregate session-start workflow
// that consumes those invariants.
//
// The evaluator is intentionally aggregate-state based so write-path decisions
// remain deterministic under replay and do not depend on projection freshness.
// Session-start reporting is split by boundary checks, aggregate indexing, and
// actionable blocker shaping so contributors can extend one concern at a time
// without re-entering a monolithic workflow file.
package readiness
