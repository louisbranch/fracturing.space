package folder

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldLevelUpApplied(state *daggerheartstate.SnapshotState, p payload.LevelUpAppliedPayload) error {
	touchCharacter(state, p.CharacterID)
	characterID := normalize.ID(p.CharacterID)
	if profile, ok := state.CharacterProfiles[characterID]; ok {
		f.applyLevelUp(&profile, p)
		state.CharacterProfiles[characterID] = profile
	}
	return nil
}

func (f *Folder) foldGoldUpdated(state *daggerheartstate.SnapshotState, p payload.GoldUpdatedPayload) error {
	touchCharacter(state, p.CharacterID)
	characterID := normalize.ID(p.CharacterID)
	if profile, ok := state.CharacterProfiles[characterID]; ok {
		profile.GoldHandfuls = p.Handfuls
		profile.GoldBags = p.Bags
		profile.GoldChests = p.Chests
		state.CharacterProfiles[characterID] = profile
	}
	return nil
}

func (f *Folder) foldDomainCardAcquired(state *daggerheartstate.SnapshotState, p payload.DomainCardAcquiredPayload) error {
	touchCharacter(state, p.CharacterID)
	characterID := normalize.ID(p.CharacterID)
	if profile, ok := state.CharacterProfiles[characterID]; ok {
		profile.DomainCardIDs = daggerheartstate.AppendUnique(profile.DomainCardIDs, p.CardID)
		state.CharacterProfiles[characterID] = profile
	}
	return nil
}

func (f *Folder) foldEquipmentSwapped(state *daggerheartstate.SnapshotState, p payload.EquipmentSwappedPayload) error {
	touchCharacter(state, p.CharacterID)
	if normalize.String(p.ItemType) == "armor" {
		characterID := normalize.ID(p.CharacterID)
		if profile, ok := state.CharacterProfiles[characterID]; ok {
			profile.EquippedArmorID = normalize.String(p.EquippedArmorID)
			if p.EvasionAfter != nil {
				profile.Evasion = *p.EvasionAfter
			}
			if p.MajorThresholdAfter != nil {
				profile.MajorThreshold = *p.MajorThresholdAfter
			}
			if p.SevereThresholdAfter != nil {
				profile.SevereThreshold = *p.SevereThresholdAfter
			}
			if p.ArmorScoreAfter != nil {
				profile.ArmorScore = *p.ArmorScoreAfter
			}
			if p.ArmorMaxAfter != nil {
				profile.ArmorMax = *p.ArmorMaxAfter
			}
			if p.SpellcastRollBonusAfter != nil {
				profile.SpellcastRollBonus = *p.SpellcastRollBonusAfter
			}
			if p.AgilityAfter != nil {
				profile.Agility = *p.AgilityAfter
			}
			if p.StrengthAfter != nil {
				profile.Strength = *p.StrengthAfter
			}
			if p.FinesseAfter != nil {
				profile.Finesse = *p.FinesseAfter
			}
			if p.InstinctAfter != nil {
				profile.Instinct = *p.InstinctAfter
			}
			if p.PresenceAfter != nil {
				profile.Presence = *p.PresenceAfter
			}
			if p.KnowledgeAfter != nil {
				profile.Knowledge = *p.KnowledgeAfter
			}
			state.CharacterProfiles[characterID] = profile
		}
		if p.ArmorAfter != nil {
			applyStatePatch(state, p.CharacterID, snapshotStatePatch{
				Armor: p.ArmorAfter,
			})
		}
	}
	if p.StressCost > 0 {
		characterID := normalize.ID(p.CharacterID)
		characterState := state.CharacterStates[characterID]
		characterState.CampaignID = state.CampaignID.String()
		characterState.CharacterID = characterID.String()
		characterState.Stress += p.StressCost
		state.CharacterStates[characterID] = characterState
	}
	return nil
}

func (f *Folder) foldConsumableUsed(state *daggerheartstate.SnapshotState, p payload.ConsumableUsedPayload) error {
	touchCharacter(state, p.CharacterID)
	return nil
}

func (f *Folder) foldConsumableAcquired(state *daggerheartstate.SnapshotState, p payload.ConsumableAcquiredPayload) error {
	touchCharacter(state, p.CharacterID)
	return nil
}

func (f *Folder) foldStatModifierChanged(state *daggerheartstate.SnapshotState, p payload.StatModifierChangedPayload) error {
	characterID := normalize.ID(p.CharacterID)
	if characterID == "" {
		return nil
	}
	state.CharacterStatModifiers[characterID] = p.Modifiers
	return nil
}
