// Package character models character entities inside a campaign.
//
// Characters are separate from participants so ownership, control, and profile
// data can evolve independently:
//   - ownership/seat links are captured via participant identity,
//   - while character-level status and metadata live in the character aggregate.
//
// The package handles character lifecycle commands, profile updates, and replay
// folds used by downstream projection and authorization checks.
package character
