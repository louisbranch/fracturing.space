package daggerheart

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// StateFactory creates Daggerheart state instances.
type StateFactory struct{}

// NewStateFactory creates a new Daggerheart state factory.
func NewStateFactory() *StateFactory {
	return &StateFactory{}
}

// NewCharacterState creates initial character state for the given character.
func (f *StateFactory) NewCharacterState(campaignID ids.CampaignID, characterID ids.CharacterID, kind string) (any, error) {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	if normalizedKind == "" {
		normalizedKind = "pc"
	}
	state := daggerheartstate.CharacterState{
		CampaignID:  strings.TrimSpace(string(campaignID)),
		CharacterID: strings.TrimSpace(string(characterID)),
		Kind:        normalizedKind,
		HP:          daggerheartstate.HPDefault,
		HPMax:       daggerheartstate.HPMaxDefault,
		Hope:        daggerheartstate.HopeDefault,
		HopeMax:     daggerheartstate.HopeMaxDefault,
		Stress:      daggerheartstate.StressDefault,
		StressMax:   daggerheartstate.StressMaxDefault,
		Armor:       daggerheartstate.ArmorDefault,
		ArmorMax:    daggerheartstate.ArmorMaxDefault,
		LifeState:   daggerheartstate.LifeStateAlive,
	}
	if normalizedKind == "npc" {
		state.Hope = 0
		state.StressMax = 0
	}
	return state, nil
}

// NewSnapshotState creates initial snapshot state for the given campaign.
// GMFear starts neutral here; first-session bootstrap owns the actual initial
// Fear seed once campaign readiness resolves the created-PC roster.
func (f *StateFactory) NewSnapshotState(campaignID ids.CampaignID) (any, error) {
	return daggerheartstate.NewSnapshotState(campaignID), nil
}
