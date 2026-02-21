package daggerheart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// profilePayload is the system-specific profile schema carried inside
// character.profile_updated events under the "daggerheart" key.
type profilePayload struct {
	Level           int                        `json:"level"`
	HpMax           int                        `json:"hp_max"`
	StressMax       int                        `json:"stress_max"`
	Evasion         int                        `json:"evasion"`
	MajorThreshold  int                        `json:"major_threshold"`
	SevereThreshold int                        `json:"severe_threshold"`
	Proficiency     int                        `json:"proficiency"`
	ArmorScore      int                        `json:"armor_score"`
	ArmorMax        int                        `json:"armor_max"`
	Experiences     []experienceProfilePayload `json:"experiences"`
	Agility         int                        `json:"agility"`
	Strength        int                        `json:"strength"`
	Finesse         int                        `json:"finesse"`
	Instinct        int                        `json:"instinct"`
	Presence        int                        `json:"presence"`
	Knowledge       int                        `json:"knowledge"`
}

type experienceProfilePayload struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}

// ApplyProfile applies a daggerheart character profile update. The raw JSON is
// the value from the system_profile map keyed by "daggerheart".
func (a *Adapter) ApplyProfile(ctx context.Context, campaignID, characterID string, profileData json.RawMessage) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("daggerheart store is not configured")
	}

	var profile profilePayload
	if err := json.Unmarshal(profileData, &profile); err != nil {
		return fmt.Errorf("decode daggerheart profile payload: %w", err)
	}

	experiences := make([]Experience, 0, len(profile.Experiences))
	for _, exp := range profile.Experiences {
		experiences = append(experiences, Experience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	level := profile.Level
	if level == 0 {
		level = PCLevelDefault
	}

	if err := ValidateProfile(
		level,
		profile.HpMax,
		profile.StressMax,
		profile.Evasion,
		profile.MajorThreshold,
		profile.SevereThreshold,
		profile.Proficiency,
		profile.ArmorScore,
		profile.ArmorMax,
		Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
		experiences,
	); err != nil {
		return fmt.Errorf("validate daggerheart profile payload: %w", err)
	}

	experienceStorage := make([]storage.DaggerheartExperience, 0, len(profile.Experiences))
	for _, exp := range profile.Experiences {
		experienceStorage = append(experienceStorage, storage.DaggerheartExperience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	return a.store.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		Level:           level,
		HpMax:           profile.HpMax,
		StressMax:       profile.StressMax,
		Evasion:         profile.Evasion,
		MajorThreshold:  profile.MajorThreshold,
		SevereThreshold: profile.SevereThreshold,
		Proficiency:     profile.Proficiency,
		ArmorScore:      profile.ArmorScore,
		ArmorMax:        profile.ArmorMax,
		Experiences:     experienceStorage,
		Agility:         profile.Agility,
		Strength:        profile.Strength,
		Finesse:         profile.Finesse,
		Instinct:        profile.Instinct,
		Presence:        profile.Presence,
		Knowledge:       profile.Knowledge,
	})
}
