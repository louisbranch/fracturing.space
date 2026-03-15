// Package gameplaystores owns the Daggerheart gameplay service's read/write
// dependency bundle.
//
// The root Daggerheart transport package depends on this sibling package for:
//   - projection-backed store wiring,
//   - startup validation of required gameplay/runtime dependencies, and
//   - projection-applier construction/caching.
//
// Keeping this bundle out of the root service package makes the remaining root
// files about gRPC behavior instead of startup/runtime support policy.
package gameplaystores
