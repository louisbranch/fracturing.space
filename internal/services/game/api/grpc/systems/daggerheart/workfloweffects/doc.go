// Package workfloweffects owns shared Daggerheart workflow side effects that
// are reused across multiple transport slices.
//
// These behaviors are not gRPC service methods themselves. They are supporting
// write-path effects triggered from outcome, recovery, and session-roll
// orchestration:
//   - repair or remove the vulnerable condition when stress crosses its max
//   - create and advance the breath countdown for adversary attack rolls
//
// Keeping them out of the root Daggerheart package makes the root package a map
// of wrapper surfaces and dependency wiring instead of a catch-all bucket for
// cross-workflow side effects.
package workfloweffects
