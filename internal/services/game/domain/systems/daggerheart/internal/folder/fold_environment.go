package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldEnvironmentEntityCreated(state *daggerheartstate.SnapshotState, p payload.EnvironmentEntityCreatedPayload) error {
	environmentEntityID := normalize.ID(p.EnvironmentEntityID)
	if environmentEntityID == "" {
		return nil
	}
	state.EnvironmentStates[environmentEntityID] = daggerheartstate.EnvironmentEntityState{
		CampaignID:          state.CampaignID,
		EnvironmentEntityID: environmentEntityID,
		EnvironmentID:       normalize.String(p.EnvironmentID),
		Name:                normalize.String(p.Name),
		Type:                normalize.String(p.Type),
		Tier:                p.Tier,
		Difficulty:          p.Difficulty,
		SessionID:           normalize.ID(p.SessionID),
		SceneID:             normalize.ID(p.SceneID),
		Notes:               normalize.String(p.Notes),
	}
	return nil
}

func (f *Folder) foldEnvironmentEntityUpdated(state *daggerheartstate.SnapshotState, p payload.EnvironmentEntityUpdatedPayload) error {
	return f.foldEnvironmentEntityCreated(state, payload.EnvironmentEntityCreatedPayload(p))
}

func (f *Folder) foldEnvironmentEntityDeleted(state *daggerheartstate.SnapshotState, p payload.EnvironmentEntityDeletedPayload) error {
	delete(state.EnvironmentStates, normalize.ID(p.EnvironmentEntityID))
	return nil
}
