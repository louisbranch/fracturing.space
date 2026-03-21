package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

type DowntimeMove = mechanics.DowntimeMove

const (
	DowntimeClearAllStress = mechanics.DowntimeClearAllStress
	DowntimeRepairAllArmor = mechanics.DowntimeRepairAllArmor
	DowntimePrepare        = mechanics.DowntimePrepare
	DowntimeWorkOnProject  = mechanics.DowntimeWorkOnProject
)

type DowntimeOptions = mechanics.DowntimeOptions
type DowntimeResult = mechanics.DowntimeResult

// ApplyDowntimeMove applies a downtime move to the character state.
func ApplyDowntimeMove(state *daggerheartstate.CharacterState, move DowntimeMove, opts DowntimeOptions) DowntimeResult {
	return mechanics.ApplyDowntimeMove(state, move, opts)
}
