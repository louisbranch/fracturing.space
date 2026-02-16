package daggerheart

import "strings"

// StateFactory creates Daggerheart state instances.
type StateFactory struct{}

// NewStateFactory creates a new Daggerheart state factory.
func NewStateFactory() *StateFactory {
	return &StateFactory{}
}

// NewCharacterState creates initial character state for the given character.
func (f *StateFactory) NewCharacterState(campaignID, characterID, kind string) (any, error) {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	if normalizedKind == "" {
		normalizedKind = "pc"
	}
	state := CharacterState{
		CampaignID:  strings.TrimSpace(campaignID),
		CharacterID: strings.TrimSpace(characterID),
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
func (f *StateFactory) NewSnapshotState(campaignID string) (any, error) {
	return SnapshotState{
		CampaignID: strings.TrimSpace(campaignID),
		GMFear:     GMFearDefault,
	}, nil
}
