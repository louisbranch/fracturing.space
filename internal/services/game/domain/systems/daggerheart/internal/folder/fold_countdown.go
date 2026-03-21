package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldCountdownCreated(state *daggerheartstate.SnapshotState, p payload.CountdownCreatedPayload) error {
	applyCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.CountdownState) {
		cs.Name = p.Name
		cs.Kind = p.Kind
		cs.Current = p.Current
		cs.Max = p.Max
		cs.Direction = p.Direction
		cs.Looping = p.Looping
		cs.Variant = p.Variant
		cs.TriggerEventType = p.TriggerEventType
		cs.LinkedCountdownID = p.LinkedCountdownID
	})
	return nil
}

func (f *Folder) foldCountdownUpdated(state *daggerheartstate.SnapshotState, p payload.CountdownUpdatedPayload) error {
	applyCountdownUpsert(state, p.CountdownID, func(cs *daggerheartstate.CountdownState) {
		cs.Current = p.Value
		if p.Looped {
			cs.Looping = true
		}
	})
	return nil
}

func (f *Folder) foldCountdownDeleted(state *daggerheartstate.SnapshotState, p payload.CountdownDeletedPayload) error {
	deleteCountdownState(state, p.CountdownID)
	return nil
}
