package folder

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldGMFearChanged(state *daggerheartstate.SnapshotState, p payload.GMFearChangedPayload) error {
	if p.Value < daggerheartstate.GMFearMin || p.Value > daggerheartstate.GMFearMax {
		return fmt.Errorf("gm fear value must be in range %d..%d", daggerheartstate.GMFearMin, daggerheartstate.GMFearMax)
	}
	state.GMFear = p.Value
	return nil
}

func (f *Folder) foldCharacterProfileReplaced(state *daggerheartstate.SnapshotState, p daggerheartstate.CharacterProfileReplacedPayload) error {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return nil
	}
	profile := p.Profile.Normalized()
	state.CharacterProfiles[characterID] = profile
	if _, exists := state.CharacterStates[characterID]; !exists {
		state.CharacterStates[characterID] = daggerheartstate.CharacterState{
			CampaignID:  normalize.ID(state.CampaignID).String(),
			CharacterID: characterID.String(),
			HP:          profile.HpMax,
			Hope:        daggerheartstate.HopeDefault,
			HopeMax:     daggerheartstate.HopeMaxDefault,
			Stress:      daggerheartstate.StressDefault,
			Armor:       profile.ArmorMax,
			LifeState:   daggerheartstate.LifeStateAlive,
		}
	}
	if profile.CompanionSheet != nil {
		state.CharacterCompanions[characterID] = daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
	} else {
		delete(state.CharacterCompanions, characterID)
	}
	return nil
}

func (f *Folder) foldCharacterProfileDeleted(state *daggerheartstate.SnapshotState, p daggerheartstate.CharacterProfileDeletedPayload) error {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return nil
	}
	delete(state.CharacterProfiles, characterID)
	delete(state.CharacterCompanions, characterID)
	return nil
}

func (f *Folder) foldCharacterStatePatched(state *daggerheartstate.SnapshotState, p payload.CharacterStatePatchedPayload) error {
	applyCharacterStatePatched(state, p)
	return nil
}

func (f *Folder) foldBeastformTransformed(state *daggerheartstate.SnapshotState, p payload.BeastformTransformedPayload) error {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return nil
	}
	nextClassState := daggerheartstate.CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = daggerheartstate.WithActiveBeastform(nextClassState, p.ActiveBeastform)
	applyStatePatch(state, p.CharacterID, snapshotStatePatch{
		Hope:       p.Hope,
		Stress:     p.Stress,
		ClassState: &nextClassState,
	})
	return nil
}

func (f *Folder) foldBeastformDropped(state *daggerheartstate.SnapshotState, p payload.BeastformDroppedPayload) error {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return nil
	}
	nextClassState := daggerheartstate.CharacterClassState{}
	if current, ok := state.CharacterClassStates[characterID]; ok {
		nextClassState = current
	}
	nextClassState = daggerheartstate.WithActiveBeastform(nextClassState, nil)
	applyStatePatch(state, p.CharacterID, snapshotStatePatch{
		ClassState: &nextClassState,
	})
	return nil
}

func (f *Folder) foldCompanionExperienceBegun(state *daggerheartstate.SnapshotState, p payload.CompanionExperienceBegunPayload) error {
	applyStatePatch(state, p.CharacterID, snapshotStatePatch{
		CompanionState: p.CompanionState,
	})
	return nil
}

func (f *Folder) foldCompanionReturned(state *daggerheartstate.SnapshotState, p payload.CompanionReturnedPayload) error {
	applyStatePatch(state, p.CharacterID, snapshotStatePatch{
		Stress:         p.Stress,
		CompanionState: p.CompanionState,
	})
	return nil
}

func (f *Folder) foldConditionChanged(state *daggerheartstate.SnapshotState, p payload.ConditionChangedPayload) error {
	applyCharacterConditionsChanged(state, p)
	return nil
}

func (f *Folder) foldLoadoutSwapped(state *daggerheartstate.SnapshotState, p payload.LoadoutSwappedPayload) error {
	applyCharacterLoadoutSwapped(state, p)
	return nil
}

func (f *Folder) foldCharacterTemporaryArmorApplied(state *daggerheartstate.SnapshotState, p payload.CharacterTemporaryArmorAppliedPayload) error {
	applyCharacterTemporaryArmorApplied(state, p)
	return nil
}
