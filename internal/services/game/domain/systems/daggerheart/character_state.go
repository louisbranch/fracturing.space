package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

const (
	ResourceHope   = mechanics.ResourceHope
	ResourceStress = mechanics.ResourceStress
	ResourceGMFear = mechanics.ResourceGMFear
	ResourceArmor  = mechanics.ResourceArmor
)

type CharacterStateConfig = mechanics.CharacterStateConfig

var (
	ErrUnknownResource      = mechanics.ErrUnknownResource
	ErrInsufficientResource = mechanics.ErrInsufficientResource
	ErrResourceAtCap        = mechanics.ErrResourceAtCap
)

// NewCharacterState creates a character state with clamped values.
func NewCharacterState(cfg CharacterStateConfig) *daggerheartstate.CharacterState {
	return mechanics.NewCharacterState(cfg)
}
