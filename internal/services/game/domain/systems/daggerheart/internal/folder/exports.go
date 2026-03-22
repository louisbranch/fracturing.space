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

// FoldCampaignCountdownUpdated is the exported standalone form of the campaign
// countdown advanced fold handler, used by root-package tests that exercise
// fold seams directly.
func FoldCampaignCountdownUpdated(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancedPayload) error {
	f := &Folder{}
	return f.foldCampaignCountdownAdvanced(state, p)
}

// FoldCountdownUpdated is the legacy generic wrapper retained temporarily for
// older tests. It maps to the scene-countdown fold path.
func FoldCountdownUpdated(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancedPayload) error {
	f := &Folder{}
	return f.foldSceneCountdownAdvanced(state, payload.SceneCountdownAdvancedPayload(p))
}

// FoldEquipmentSwapped is the exported standalone form of the equipment
// swapped fold handler, used by root-package tests that exercise fold seams
// directly.
func FoldEquipmentSwapped(state *daggerheartstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	f := &Folder{}
	return f.foldEquipmentSwapped(state, p)
}
