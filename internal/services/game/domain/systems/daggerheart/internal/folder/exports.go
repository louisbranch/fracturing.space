package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

// FoldGMFearChanged is the exported standalone form of the GM fear fold
// handler, used by root-package test aliases.
func FoldGMFearChanged(state *snapstate.SnapshotState, p payload.GMFearChangedPayload) error {
	f := &Folder{}
	return f.foldGMFearChanged(state, p)
}

// FoldCountdownUpdated is the exported standalone form of the countdown
// updated fold handler, used by root-package test aliases.
func FoldCountdownUpdated(state *snapstate.SnapshotState, p payload.CountdownUpdatedPayload) error {
	f := &Folder{}
	return f.foldCountdownUpdated(state, p)
}

// FoldEquipmentSwapped is the exported standalone form of the equipment
// swapped fold handler, used by root-package test aliases.
func FoldEquipmentSwapped(state *snapstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	f := &Folder{}
	return f.foldEquipmentSwapped(state, p)
}
