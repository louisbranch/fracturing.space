package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldDamageApplied(state *daggerheartstate.SnapshotState, p payload.DamageAppliedPayload) error {
	applyDamageApplied(state, p.CharacterID, p.Hp, p.Stress, p.Armor)
	return nil
}

func (f *Folder) foldAdversaryDamageApplied(state *daggerheartstate.SnapshotState, p payload.AdversaryDamageAppliedPayload) error {
	applyAdversaryDamage(state, p.AdversaryID, p.Hp, p.Armor)
	return nil
}

func (f *Folder) foldDowntimeMoveApplied(state *daggerheartstate.SnapshotState, p payload.DowntimeMoveAppliedPayload) error {
	targetID := p.TargetCharacterID
	if normalize.ID(targetID) == "" {
		targetID = p.ActorCharacterID
	}
	if normalize.ID(targetID) == "" {
		return nil
	}
	applyStatePatch(state, targetID, snapshotStatePatch{
		HP:     p.HP,
		Hope:   p.Hope,
		Stress: p.Stress,
		Armor:  p.Armor,
	})
	return nil
}

func (f *Folder) foldAdversaryConditionChanged(state *daggerheartstate.SnapshotState, p payload.AdversaryConditionChangedPayload) error {
	applyAdversaryConditionsChanged(state, p.AdversaryID, p.Conditions)
	return nil
}
