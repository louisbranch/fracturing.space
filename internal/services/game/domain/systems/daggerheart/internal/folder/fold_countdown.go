package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldSceneCountdownCreated(state *daggerheartstate.SnapshotState, p payload.SceneCountdownCreatedPayload) error {
	applySceneCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.SceneCountdownState) {
		cs.SessionID = p.SessionID
		cs.SceneID = p.SceneID
		cs.Name = p.Name
		cs.Tone = p.Tone
		cs.AdvancementPolicy = p.AdvancementPolicy
		cs.StartingValue = p.StartingValue
		cs.RemainingValue = p.RemainingValue
		cs.LoopBehavior = p.LoopBehavior
		cs.Status = p.Status
		cs.LinkedCountdownID = p.LinkedCountdownID
		if p.StartingRoll != nil {
			cs.StartingRoll = &rules.CountdownStartingRoll{Min: p.StartingRoll.Min, Max: p.StartingRoll.Max, Value: p.StartingRoll.Value}
		} else {
			cs.StartingRoll = nil
		}
	})
	return nil
}

func (f *Folder) foldCampaignCountdownCreated(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownCreatedPayload) error {
	applyCampaignCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.CampaignCountdownState) {
		cs.Name = p.Name
		cs.Tone = p.Tone
		cs.AdvancementPolicy = p.AdvancementPolicy
		cs.StartingValue = p.StartingValue
		cs.RemainingValue = p.RemainingValue
		cs.LoopBehavior = p.LoopBehavior
		cs.Status = p.Status
		cs.LinkedCountdownID = p.LinkedCountdownID
		if p.StartingRoll != nil {
			cs.StartingRoll = &rules.CountdownStartingRoll{Min: p.StartingRoll.Min, Max: p.StartingRoll.Max, Value: p.StartingRoll.Value}
		} else {
			cs.StartingRoll = nil
		}
	})
	return nil
}

func (f *Folder) foldSceneCountdownAdvanced(state *daggerheartstate.SnapshotState, p payload.SceneCountdownAdvancedPayload) error {
	applySceneCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.SceneCountdownState) {
		cs.RemainingValue = p.AfterRemaining
		cs.Status = p.StatusAfter
	})
	return nil
}

func (f *Folder) foldCampaignCountdownAdvanced(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancedPayload) error {
	applyCampaignCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.CampaignCountdownState) {
		cs.RemainingValue = p.AfterRemaining
		cs.Status = p.StatusAfter
	})
	return nil
}

func (f *Folder) foldSceneCountdownTriggerResolved(state *daggerheartstate.SnapshotState, p payload.SceneCountdownTriggerResolvedPayload) error {
	applySceneCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.SceneCountdownState) {
		cs.StartingValue = p.StartingValueAfter
		cs.RemainingValue = p.RemainingValueAfter
		cs.Status = p.StatusAfter
	})
	return nil
}

func (f *Folder) foldCampaignCountdownTriggerResolved(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownTriggerResolvedPayload) error {
	applyCampaignCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.CampaignCountdownState) {
		cs.StartingValue = p.StartingValueAfter
		cs.RemainingValue = p.RemainingValueAfter
		cs.Status = p.StatusAfter
	})
	return nil
}

func (f *Folder) foldSceneCountdownDeleted(state *daggerheartstate.SnapshotState, p payload.SceneCountdownDeletedPayload) error {
	deleteSceneCountdownState(state, p.CountdownID)
	return nil
}

func (f *Folder) foldCampaignCountdownDeleted(state *daggerheartstate.SnapshotState, p payload.CampaignCountdownDeletedPayload) error {
	deleteCampaignCountdownState(state, p.CountdownID)
	return nil
}
