// Package decide contains reusable decision-flow helpers for command deciders.
//
// All helpers follow the same pipeline: unmarshal → validate → marshal → emit.
// Choose the variant that matches your decider's needs:
//
//   - [DecideFunc]: stateless, single-event emission. Use when the decider does
//     not need snapshot state and emits one event (e.g. campaign.create).
//
//   - [DecideFuncWithState]: snapshot-aware, single-event emission. Use when the
//     decider needs aggregate snapshot state for before-value checks or idempotency
//     guards but the event payload matches the command payload shape.
//
//   - [DecideFuncTransform]: snapshot-aware with payload shape change. Use when the
//     command payload type differs from the event payload type (e.g. a "set" command
//     records a "changed" event with before/after fields).
//
//   - [DecideFuncMulti]: snapshot-aware, multi-event emission. Use when a single
//     command produces multiple events atomically (e.g. batch operations).
//
// For deciders with complex branching, conditional event emission, or cross-entity
// orchestration that doesn't fit these patterns, use a raw switch in the decider
// function directly.
package decide
