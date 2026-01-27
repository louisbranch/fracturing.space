// Package domain provides the core rules evaluation and dice mechanics for the Duality engine.
//
// The Duality system is a 2d12 dice mechanic (Daggerheart-compatible) where one die
// represents Hope and the other represents Fear.
//
// # Core Mechanics
//
// An Action Roll consists of rolling two 12-sided dice (Hope and Fear) and adding
// a modifier. The total is compared against a difficulty threshold.
//   - Total = Hope + Fear + Modifier
//   - Success: Total >= Difficulty
//   - Critical Success: Hope and Fear dice show the same value (e.g., double 6s).
//     A critical success always succeeds, regardless of the difficulty.
//
// # Evaluation Outcomes
//
// The result of a roll is categorized into several outcomes:
//   - Critical Success: Matching dice.
//   - Success with Hope/Fear: Total meets difficulty, Hope/Fear is higher.
//   - Failure with Hope/Fear: Total below difficulty, Hope/Fear is higher.
//   - Roll with Hope/Fear: Deterministic evaluation without a difficulty threshold.
//
// # Features
//
// This package includes utilities for:
//   - Dice Rolling: Deterministic seeds for reproducible rolls.
//   - Outcome Evaluation: Logic for resolving action rolls and deterministic comparisons.
//   - Probability: Calculating outcome counts across the entire 2d12 space.
//   - Explanations: Generating human-readable steps for how an outcome was reached.
package domain
