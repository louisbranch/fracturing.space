// Package recoverytransport owns the Daggerheart recovery and life-state
// mutation transport endpoints.
//
// This slice groups the gameplay mutations that share campaign/session gate
// checks, character or snapshot reloads, seeded resolution, and system-command
// emission: rest, downtime, temporary armor, loadout swaps, death moves, and
// blaze-of-glory resolution.
package recoverytransport
