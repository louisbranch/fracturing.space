package daggerheart

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func applyLevelUpToCharacterProfile(profile *CharacterProfile, payload LevelUpAppliedPayload) {
	if profile == nil {
		return
	}

	profile.Level = payload.Level
	profile.MajorThreshold += payload.ThresholdDelta
	profile.SevereThreshold += payload.ThresholdDelta * 2

	for _, adv := range payload.Advancements {
		switch adv.Type {
		case "trait_increase":
			applyCharacterProfileTraitIncrease(profile, adv.Trait)
		case "add_hp_slots":
			profile.HpMax++
		case "add_stress_slots":
			profile.StressMax++
		case "increase_evasion":
			profile.Evasion++
		case "increase_proficiency":
			profile.Proficiency++
		case "increase_experience":
			// Experience additions are content-level; no profile field change needed.
		case "domain_card":
			if adv.DomainCardID != "" {
				profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, strings.TrimSpace(adv.DomainCardID))
			}
		case "upgraded_subclass":
			if adv.SubclassCardID != "" {
				profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, strings.TrimSpace(adv.SubclassCardID))
			}
		}
	}

	if payload.NewDomainCardID != "" {
		profile.DomainCardIDs = appendUnique(profile.DomainCardIDs, strings.TrimSpace(payload.NewDomainCardID))
	}
}

func applyCharacterProfileTraitIncrease(profile *CharacterProfile, trait string) {
	switch strings.TrimSpace(trait) {
	case "agility":
		profile.Agility++
	case "strength":
		profile.Strength++
	case "finesse":
		profile.Finesse++
	case "instinct":
		profile.Instinct++
	case "presence":
		profile.Presence++
	case "knowledge":
		profile.Knowledge++
	}
}

func applyProfileTraitIncrease(profile *projectionstore.DaggerheartCharacterProfile, trait string) {
	if profile == nil {
		return
	}
	typed := CharacterProfileFromStorage(*profile)
	applyCharacterProfileTraitIncrease(&typed, trait)
	*profile = typed.ToStorage(profile.CampaignID, profile.CharacterID)
}
