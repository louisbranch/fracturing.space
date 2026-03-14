package daggerheart

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
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
	state := CharacterState{
		CampaignID:  strings.TrimSpace(string(campaignID)),
		CharacterID: strings.TrimSpace(string(characterID)),
		Kind:        normalizedKind,
		HP:          HPDefault,
		HPMax:       HPMaxDefault,
		Hope:        HopeDefault,
		HopeMax:     HopeMaxDefault,
		Stress:      StressDefault,
		StressMax:   StressMaxDefault,
		Armor:       ArmorDefault,
		ArmorMax:    ArmorMaxDefault,
		LifeState:   LifeStateAlive,
	}
	if normalizedKind == "npc" {
		state.Hope = 0
		state.StressMax = 0
	}
	return state, nil
}

// NewSnapshotState creates initial snapshot state for the given campaign.
func (f *StateFactory) NewSnapshotState(campaignID ids.CampaignID) (any, error) {
	return SnapshotState{
		CampaignID:        ids.CampaignID(strings.TrimSpace(string(campaignID))),
		GMFear:            GMFearDefault,
		CharacterProfiles: make(map[ids.CharacterID]CharacterProfile),
		CharacterStates:   make(map[ids.CharacterID]CharacterState),
		AdversaryStates:   make(map[ids.AdversaryID]AdversaryState),
		CountdownStates:   make(map[ids.CountdownID]CountdownState),
	}, nil
}
