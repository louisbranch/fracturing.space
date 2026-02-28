// Package readiness evaluates campaign/session readiness invariants that gate
// session start decisions.
//
// The evaluator is intentionally aggregate-state based so write-path decisions
// remain deterministic under replay and do not depend on projection freshness.
package readiness
