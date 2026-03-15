package creationworkflow

import (
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
)

func validateProfile(profile projectionstore.DaggerheartCharacterProfile) error {
	experiences := make([]daggerheartprofile.Experience, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	return daggerheartprofile.Validate(
		profile.Level,
		profile.HpMax,
		profile.StressMax,
		profile.Evasion,
		profile.MajorThreshold,
		profile.SevereThreshold,
		profile.Proficiency,
		profile.ArmorScore,
		profile.ArmorMax,
		daggerheartprofile.Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
		experiences,
	)
}

func defaultProfileForCharacter(campaignID string, kind character.Kind) projectionstore.DaggerheartCharacterProfile {
	profile := projectionstore.DaggerheartCharacterProfile{
		CampaignID: campaignID,
	}
	return ensureProfileDefaults(profile, kind)
}

func ensureProfileDefaults(profile projectionstore.DaggerheartCharacterProfile, kind character.Kind) projectionstore.DaggerheartCharacterProfile {
	kindLabel := "PC"
	if kind == character.KindNPC {
		kindLabel = "NPC"
	}
	defaults := daggerheartprofile.GetDefaults(kindLabel)

	if profile.Level == 0 {
		profile.Level = defaults.Level
	}
	if profile.HpMax == 0 {
		profile.HpMax = defaults.HpMax
	}
	if profile.StressMax == 0 {
		profile.StressMax = defaults.StressMax
	}
	if profile.Evasion == 0 {
		profile.Evasion = defaults.Evasion
	}
	if profile.Proficiency == 0 {
		profile.Proficiency = defaults.Proficiency
	}
	if profile.ArmorMax == 0 {
		profile.ArmorMax = defaults.ArmorMax
	}
	if profile.MajorThreshold == 0 && profile.SevereThreshold == 0 {
		profile.MajorThreshold, profile.SevereThreshold = daggerheartprofile.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			defaults.MajorThreshold,
			defaults.SevereThreshold,
		)
	}
	return profile
}
