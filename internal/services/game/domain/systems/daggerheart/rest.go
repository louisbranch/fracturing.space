package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/mechanics"

type RestType = mechanics.RestType

const (
	RestTypeShort = mechanics.RestTypeShort
	RestTypeLong  = mechanics.RestTypeLong
)

type RestState = mechanics.RestState
type RestOutcome = mechanics.RestOutcome

var (
	ErrInvalidRestSequence = mechanics.ErrInvalidRestSequence
)

// ResolveRestOutcome applies rest rules and consequences.
func ResolveRestOutcome(state RestState, restType RestType, interrupted bool, seed int64, partySize int) (RestOutcome, error) {
	return mechanics.ResolveRestOutcome(state, restType, interrupted, seed, partySize)
}
