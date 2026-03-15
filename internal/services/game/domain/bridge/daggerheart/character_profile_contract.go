package daggerheart

import (
	"fmt"
	"strings"

	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// CharacterProfileExperience captures one named Daggerheart experience entry.
type CharacterProfileExperience struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}

// CharacterProfile is the authoritative Daggerheart character profile contract
// used by typed system-owned profile commands/events and aggregate state.
type CharacterProfile struct {
	Level                int                          `json:"level"`
	HpMax                int                          `json:"hp_max"`
	StressMax            int                          `json:"stress_max"`
	Evasion              int                          `json:"evasion"`
	MajorThreshold       int                          `json:"major_threshold"`
	SevereThreshold      int                          `json:"severe_threshold"`
	Proficiency          int                          `json:"proficiency"`
	ArmorScore           int                          `json:"armor_score"`
	ArmorMax             int                          `json:"armor_max"`
	Experiences          []CharacterProfileExperience `json:"experiences,omitempty"`
	Agility              int                          `json:"agility"`
	Strength             int                          `json:"strength"`
	Finesse              int                          `json:"finesse"`
	Instinct             int                          `json:"instinct"`
	Presence             int                          `json:"presence"`
	Knowledge            int                          `json:"knowledge"`
	ClassID              string                       `json:"class_id,omitempty"`
	SubclassID           string                       `json:"subclass_id,omitempty"`
	AncestryID           string                       `json:"ancestry_id,omitempty"`
	CommunityID          string                       `json:"community_id,omitempty"`
	TraitsAssigned       bool                         `json:"traits_assigned"`
	DetailsRecorded      bool                         `json:"details_recorded"`
	StartingWeaponIDs    []string                     `json:"starting_weapon_ids,omitempty"`
	StartingArmorID      string                       `json:"starting_armor_id,omitempty"`
	StartingPotionItemID string                       `json:"starting_potion_item_id,omitempty"`
	Background           string                       `json:"background,omitempty"`
	Description          string                       `json:"description,omitempty"`
	DomainCardIDs        []string                     `json:"domain_card_ids,omitempty"`
	Connections          string                       `json:"connections,omitempty"`
	GoldHandfuls         int                          `json:"gold_handfuls,omitempty"`
	GoldBags             int                          `json:"gold_bags,omitempty"`
	GoldChests           int                          `json:"gold_chests,omitempty"`
}

// CharacterProfileReplacePayload captures the payload for
// sys.daggerheart.character_profile.replace commands.
type CharacterProfileReplacePayload struct {
	CharacterID ids.CharacterID  `json:"character_id"`
	Profile     CharacterProfile `json:"profile"`
}

// CharacterProfileReplacedPayload captures the payload for
// sys.daggerheart.character_profile.replaced events.
type CharacterProfileReplacedPayload = CharacterProfileReplacePayload

// CharacterProfileDeletePayload captures the payload for
// sys.daggerheart.character_profile.delete commands.
type CharacterProfileDeletePayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Reason      string          `json:"reason,omitempty"`
}

// CharacterProfileDeletedPayload captures the payload for
// sys.daggerheart.character_profile.deleted events.
type CharacterProfileDeletedPayload = CharacterProfileDeletePayload

// Normalized applies defaulted profile values before validation, replay, or
// projection so all write paths persist the same authoritative shape.
func (p CharacterProfile) Normalized() CharacterProfile {
	normalized := p
	if normalized.Level == 0 {
		normalized.Level = daggerheartprofile.PCLevelDefault
	}
	return normalized
}

// Validate checks that the profile is structurally valid for projection and replay.
func (p CharacterProfile) Validate() error {
	normalized := p.Normalized()

	experiences := make([]daggerheartprofile.Experience, 0, len(normalized.Experiences))
	for _, exp := range normalized.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	if err := daggerheartprofile.Validate(
		normalized.Level,
		normalized.HpMax,
		normalized.StressMax,
		normalized.Evasion,
		normalized.MajorThreshold,
		normalized.SevereThreshold,
		normalized.Proficiency,
		normalized.ArmorScore,
		normalized.ArmorMax,
		daggerheartprofile.Traits{
			Agility:   normalized.Agility,
			Strength:  normalized.Strength,
			Finesse:   normalized.Finesse,
			Instinct:  normalized.Instinct,
			Presence:  normalized.Presence,
			Knowledge: normalized.Knowledge,
		},
		experiences,
	); err != nil {
		return fmt.Errorf("validate daggerheart character profile: %w", err)
	}
	return nil
}

// ToStorage converts the typed profile contract into the projected storage form.
func (p CharacterProfile) ToStorage(campaignID, characterID string) projectionstore.DaggerheartCharacterProfile {
	normalized := p.Normalized()

	experiences := make([]projectionstore.DaggerheartExperience, 0, len(normalized.Experiences))
	for _, exp := range normalized.Experiences {
		experiences = append(experiences, projectionstore.DaggerheartExperience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:           strings.TrimSpace(campaignID),
		CharacterID:          strings.TrimSpace(characterID),
		Level:                normalized.Level,
		HpMax:                normalized.HpMax,
		StressMax:            normalized.StressMax,
		Evasion:              normalized.Evasion,
		MajorThreshold:       normalized.MajorThreshold,
		SevereThreshold:      normalized.SevereThreshold,
		Proficiency:          normalized.Proficiency,
		ArmorScore:           normalized.ArmorScore,
		ArmorMax:             normalized.ArmorMax,
		Experiences:          experiences,
		ClassID:              normalized.ClassID,
		SubclassID:           normalized.SubclassID,
		AncestryID:           normalized.AncestryID,
		CommunityID:          normalized.CommunityID,
		TraitsAssigned:       normalized.TraitsAssigned,
		DetailsRecorded:      normalized.DetailsRecorded,
		StartingWeaponIDs:    append([]string(nil), normalized.StartingWeaponIDs...),
		StartingArmorID:      normalized.StartingArmorID,
		StartingPotionItemID: normalized.StartingPotionItemID,
		Background:           normalized.Background,
		Description:          normalized.Description,
		DomainCardIDs:        append([]string(nil), normalized.DomainCardIDs...),
		Connections:          normalized.Connections,
		GoldHandfuls:         normalized.GoldHandfuls,
		GoldBags:             normalized.GoldBags,
		GoldChests:           normalized.GoldChests,
		Agility:              normalized.Agility,
		Strength:             normalized.Strength,
		Finesse:              normalized.Finesse,
		Instinct:             normalized.Instinct,
		Presence:             normalized.Presence,
		Knowledge:            normalized.Knowledge,
	}
}

// CharacterProfileFromStorage converts projected storage state into the typed
// Daggerheart profile contract.
func CharacterProfileFromStorage(profile projectionstore.DaggerheartCharacterProfile) CharacterProfile {
	experiences := make([]CharacterProfileExperience, 0, len(profile.Experiences))
	for _, exp := range profile.Experiences {
		experiences = append(experiences, CharacterProfileExperience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	return CharacterProfile{
		Level:                profile.Level,
		HpMax:                profile.HpMax,
		StressMax:            profile.StressMax,
		Evasion:              profile.Evasion,
		MajorThreshold:       profile.MajorThreshold,
		SevereThreshold:      profile.SevereThreshold,
		Proficiency:          profile.Proficiency,
		ArmorScore:           profile.ArmorScore,
		ArmorMax:             profile.ArmorMax,
		Experiences:          experiences,
		Agility:              profile.Agility,
		Strength:             profile.Strength,
		Finesse:              profile.Finesse,
		Instinct:             profile.Instinct,
		Presence:             profile.Presence,
		Knowledge:            profile.Knowledge,
		ClassID:              profile.ClassID,
		SubclassID:           profile.SubclassID,
		AncestryID:           profile.AncestryID,
		CommunityID:          profile.CommunityID,
		TraitsAssigned:       profile.TraitsAssigned,
		DetailsRecorded:      profile.DetailsRecorded,
		StartingWeaponIDs:    append([]string(nil), profile.StartingWeaponIDs...),
		StartingArmorID:      profile.StartingArmorID,
		StartingPotionItemID: profile.StartingPotionItemID,
		Background:           profile.Background,
		Description:          profile.Description,
		DomainCardIDs:        append([]string(nil), profile.DomainCardIDs...),
		Connections:          profile.Connections,
		GoldHandfuls:         profile.GoldHandfuls,
		GoldBags:             profile.GoldBags,
		GoldChests:           profile.GoldChests,
	}
}

// CreationProfile converts the full profile contract into the subset used by
// creation workflow progress and readiness evaluation.
func (p CharacterProfile) CreationProfile() CreationProfile {
	normalized := p.Normalized()

	experiences := make([]daggerheartprofile.Experience, 0, len(normalized.Experiences))
	for _, exp := range normalized.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	return CreationProfile{
		ClassID:        normalized.ClassID,
		SubclassID:     normalized.SubclassID,
		AncestryID:     normalized.AncestryID,
		CommunityID:    normalized.CommunityID,
		TraitsAssigned: normalized.TraitsAssigned,
		Traits: daggerheartprofile.Traits{
			Agility:   normalized.Agility,
			Strength:  normalized.Strength,
			Finesse:   normalized.Finesse,
			Instinct:  normalized.Instinct,
			Presence:  normalized.Presence,
			Knowledge: normalized.Knowledge,
		},
		DetailsRecorded:      normalized.DetailsRecorded,
		Level:                normalized.Level,
		HpMax:                normalized.HpMax,
		StressMax:            normalized.StressMax,
		Evasion:              normalized.Evasion,
		StartingWeaponIDs:    append([]string(nil), normalized.StartingWeaponIDs...),
		StartingArmorID:      normalized.StartingArmorID,
		StartingPotionItemID: normalized.StartingPotionItemID,
		Background:           normalized.Background,
		Description:          normalized.Description,
		Experiences:          experiences,
		DomainCardIDs:        append([]string(nil), normalized.DomainCardIDs...),
		Connections:          normalized.Connections,
	}
}
