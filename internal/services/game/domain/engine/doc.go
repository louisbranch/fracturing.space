// Package engine wires command validation, gate checks, lifecycle policy,
// decision routing, event append, and replay-backed state loading for domain
// command execution.
//
// This package is the runtime seam between immutable domain contracts and
// transport handlers: it validates intent, applies domain policy, persists events,
// and returns a replayable decision/result for downstream handlers.
//
// Session start remains the one intentional cross-aggregate exception inside
// the core write path. CoreDecider reaches that behavior through a
// readiness-owned workflow seam so the engine stays focused on routing rather
// than owning readiness and campaign-activation orchestration directly.
package engine
