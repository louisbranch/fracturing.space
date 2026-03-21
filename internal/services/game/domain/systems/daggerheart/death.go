package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"

const (
	DeathMoveBlazeOfGlory = mechanics.DeathMoveBlazeOfGlory
	DeathMoveAvoidDeath   = mechanics.DeathMoveAvoidDeath
	DeathMoveRiskItAll    = mechanics.DeathMoveRiskItAll
)

type DeathMoveInput = mechanics.DeathMoveInput
type DeathMoveOutcome = mechanics.DeathMoveOutcome

// NormalizeDeathMove validates and normalizes a death move value.
func NormalizeDeathMove(value string) (string, error) {
	return mechanics.NormalizeDeathMove(value)
}

// NormalizeLifeState validates and normalizes a life state value.
func NormalizeLifeState(value string) (string, error) {
	return mechanics.NormalizeLifeState(value)
}

// ResolveDeathMove applies the Daggerheart death move rules.
func ResolveDeathMove(input DeathMoveInput) (DeathMoveOutcome, error) {
	return mechanics.ResolveDeathMove(input)
}
