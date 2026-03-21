package daggerheart

import (
	"strings"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func applyLevelUpToCharacterProfile(profile *daggerheartstate.CharacterProfile, payload daggerheartpayload.LevelUpAppliedPayload) {
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
				profile.DomainCardIDs = daggerheartstate.AppendUnique(profile.DomainCardIDs, strings.TrimSpace(adv.DomainCardID))
			}
		}
	}

	for _, reward := range payload.Rewards {
		if strings.TrimSpace(reward.Type) == "domain_card" && reward.DomainCardID != "" {
			profile.DomainCardIDs = daggerheartstate.AppendUnique(profile.DomainCardIDs, strings.TrimSpace(reward.DomainCardID))
		}
	}
	if len(payload.SubclassTracksAfter) > 0 {
		profile.SubclassTracks = append([]daggerheartstate.CharacterSubclassTrack(nil), payload.SubclassTracksAfter...)
	}
	profile.HpMax += payload.SubclassHpMaxDelta
	profile.StressMax += payload.SubclassStressMaxDelta
	profile.Evasion += payload.SubclassEvasionDelta
	profile.MajorThreshold += payload.SubclassMajorThresholdDelta
	profile.SevereThreshold += payload.SubclassSevereThresholdDelta
}

func applyCharacterProfileTraitIncrease(profile *daggerheartstate.CharacterProfile, trait string) {
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
