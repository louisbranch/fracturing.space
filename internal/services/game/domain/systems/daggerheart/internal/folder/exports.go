package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// FoldGMFearChanged is the exported standalone form of the GM fear fold
// handler, used by root-package tests that exercise fold seams directly.
func FoldGMFearChanged(state *daggerheartstate.SnapshotState, p payload.GMFearChangedPayload) error {
	f := &Folder{}
	return f.foldGMFearChanged(state, p)
}

// FoldCountdownUpdated is the exported standalone form of the countdown
// updated fold handler, used by root-package tests that exercise fold seams
// directly.
func FoldCountdownUpdated(state *daggerheartstate.SnapshotState, p payload.CountdownUpdatedPayload) error {
	f := &Folder{}
	return f.foldCountdownUpdated(state, p)
}

// FoldEquipmentSwapped is the exported standalone form of the equipment
// swapped fold handler, used by root-package tests that exercise fold seams
// directly.
func FoldEquipmentSwapped(state *daggerheartstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	f := &Folder{}
	return f.foldEquipmentSwapped(state, p)
}
