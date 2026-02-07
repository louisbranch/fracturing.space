// Package daggerheart implements the Daggerheart game system.
//
// Daggerheart uses "Duality dice" - a pair of d12s called Hope and Fear.
// The system determines outcomes based on which die rolls higher and
// whether the total meets a difficulty target.
//
// # Duality Dice Mechanics
//
// Every action roll uses 2d12 (Hope + Fear):
//   - If Hope > Fear: roll/succeed "with hope" (player gets narrative advantage)
//   - If Fear > Hope: roll/succeed "with fear" (GM gets narrative opportunity)
//   - If Hope == Fear: Critical Success (always succeeds, bonus effects)
//
// # Outcomes
//
// Without difficulty: ROLL_WITH_HOPE, ROLL_WITH_FEAR, or CRITICAL_SUCCESS
// With difficulty: SUCCESS_WITH_HOPE, SUCCESS_WITH_FEAR, FAILURE_WITH_HOPE,
// FAILURE_WITH_FEAR, or CRITICAL_SUCCESS
//
// # Resources
//
// Characters track Hope (a spendable resource) and may track Stress.
// The GM tracks Fear as a campaign-level resource.
//
// # Package Structure
//
//   - domain/: Core Daggerheart mechanics (outcomes, probability, rules)
//   - service/: gRPC service handlers (uses legacy duality.v1 protos for now)
package daggerheart
