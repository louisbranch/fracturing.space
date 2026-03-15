// Package aggregate composes aggregate folds across domain areas for replay and command execution.
//
// # Systems map and type safety
//
// The Systems field (map[module.Key]any) holds per-game-system state keyed by
// module.Key (system ID + version). Values are typed as any because the map is
// heterogeneous: each game system defines its own state type, and this package
// cannot know all system types at compile time.
//
// Type safety is recovered at runtime via [AssertState], which extracts and
// validates typed state from the any value, returning a clear error when the
// stored type does not match the caller's expectation.
//
// This is an intentional trade-off. Go generics cannot express a heterogeneous
// map where each key maps to a different concrete type, so the runtime
// assertion is the narrowest practical escape hatch.
package aggregate
