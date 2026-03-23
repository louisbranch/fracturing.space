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

// FoldSceneCountdownAdvanced is the exported standalone form of the scene
// countdown advanced fold handler, used by root-package tests that exercise
// fold seams directly.
func FoldSceneCountdownAdvanced(state *daggerheartstate.SnapshotState, p payload.SceneCountdownAdvancedPayload) error {
	f := &Folder{}
	return f.foldSceneCountdownAdvanced(state, p)
}

// FoldCampaignCountdownAdvanced is the exported standalone form of the
// campaign countdown advanced fold handler, used by root-package tests that
// exercise fold seams directly.
func FoldCampaignCountdownAdvanced(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancedPayload) error {
	f := &Folder{}
	return f.foldCampaignCountdownAdvanced(state, p)
}

// FoldEquipmentSwapped is the exported standalone form of the equipment
// swapped fold handler, used by root-package tests that exercise fold seams
// directly.
func FoldEquipmentSwapped(state *daggerheartstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	f := &Folder{}
	return f.foldEquipmentSwapped(state, p)
}
