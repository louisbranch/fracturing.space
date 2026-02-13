// Package core provides generic RPG mechanics primitives.
//
// The core package contains system-agnostic functionality that can be used by
// any tabletop RPG game system. It is organized into subpackages:
//
//   - dice: Dice rolling primitives (NdM rolls, deterministic seeding)
//   - check: Difficulty check primitives (success/failure, margin calculation)
//   - random: Cryptographic seed helpers
//
// Game-system-specific mechanics (like Daggerheart's Duality dice, D&D 5e's
// advantage/disadvantage, or Vampire's success-counting pools) are built on
// top of these primitives in the internal/services/game/domain/systems/ packages.
//
// # Design Philosophy
//
// Core mechanics are intentionally minimal and unopinionated. They provide
// building blocks without imposing any particular game system's interpretation.
// For example, the dice package rolls dice but doesn't know about critical
// hits - that's the job of the game system layer.
package core
