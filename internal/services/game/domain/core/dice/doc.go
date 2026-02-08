// Package dice provides generic dice rolling primitives.
//
// This package contains system-agnostic dice rolling functionality that can be
// used by any game system. It provides:
//
//   - Basic die rolling with configurable RNG
//   - Dice specifications (NdM notation)
//   - Deterministic rolling with seed support
//   - Dice pool aggregation
//
// Game-system-specific mechanics (like Daggerheart's Duality dice interpretation)
// are built on top of these primitives in the systems/ packages.
package dice
