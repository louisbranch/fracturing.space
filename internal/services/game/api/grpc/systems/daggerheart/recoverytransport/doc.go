// Package recoverytransport owns the Daggerheart recovery and life-state
// mutation transport endpoints.
//
// This slice groups the gameplay mutations that share campaign/session gate
// checks, character or snapshot reloads, seeded resolution, and system-command
// emission: rest, downtime, temporary armor, loadout swaps, death moves, and
// blaze-of-glory resolution. The package now reads by workflow file rather
// than one mixed handler: start with the specific `handler_*.go` entrypoint,
// then use `handler_helpers.go` only for shared dependency checks.
package recoverytransport
