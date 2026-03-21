package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldAdversaryCreated(state *daggerheartstate.SnapshotState, p payload.AdversaryCreatePayload) error {
	applyAdversaryCreated(state, p)
	return nil
}

func (f *Folder) foldAdversaryUpdated(state *daggerheartstate.SnapshotState, p payload.AdversaryUpdatePayload) error {
	applyAdversaryUpdated(state, p)
	return nil
}

func (f *Folder) foldAdversaryDeleted(state *daggerheartstate.SnapshotState, p payload.AdversaryDeletedPayload) error {
	delete(state.AdversaryStates, normalize.ID(p.AdversaryID))
	return nil
}
