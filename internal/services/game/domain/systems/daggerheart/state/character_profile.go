package state

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

const (
	// CompanionSheetDefaultEvasion is the starting companion evasion.
	CompanionSheetDefaultEvasion = 10
	// CompanionSheetDefaultAttackRange is the fixed starting companion range.
	CompanionSheetDefaultAttackRange = "melee"
	// CompanionSheetDefaultDamageDieSides is the fixed starting companion damage die.
	CompanionSheetDefaultDamageDieSides = 6
	// CompanionSheetExperienceModifier is the fixed creation-time experience modifier.
	CompanionSheetExperienceModifier = 2
	// CompanionDamageTypePhysical is the physical companion damage type.
	CompanionDamageTypePhysical = "physical"
	// CompanionDamageTypeMagic is the magic companion damage type.
	CompanionDamageTypeMagic = "magic"
	// SubclassCreationRequirementCompanionSheet requires a companion sheet.
	SubclassCreationRequirementCompanionSheet = "companion_sheet_required"
	// SubclassTrackOriginPrimary marks the main class/subclass progression.
	SubclassTrackOriginPrimary = "primary"
	// SubclassTrackOriginMulticlass marks a multiclass subclass track.
	SubclassTrackOriginMulticlass = "multiclass"
	// SubclassTrackRankFoundation unlocks foundation features.
	SubclassTrackRankFoundation = "foundation"
	// SubclassTrackRankSpecialization unlocks specialization features.
	SubclassTrackRankSpecialization = "specialization"
	// SubclassTrackRankMastery unlocks mastery features.
	SubclassTrackRankMastery = "mastery"
)

// CharacterProfileExperience captures one named Daggerheart experience entry.
type CharacterProfileExperience struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}

// CharacterHeritage captures the structured ancestry/community choice stored on
// a Daggerheart character profile.
type CharacterHeritage struct {
	AncestryLabel           string `json:"ancestry_label,omitempty"`
	FirstFeatureAncestryID  string `json:"first_feature_ancestry_id,omitempty"`
	FirstFeatureID          string `json:"first_feature_id,omitempty"`
	SecondFeatureAncestryID string `json:"second_feature_ancestry_id,omitempty"`
	SecondFeatureID         string `json:"second_feature_id,omitempty"`
	CommunityID             string `json:"community_id,omitempty"`
}

// Validate checks that the structured heritage selection is internally valid.
func (h CharacterHeritage) Validate() error {
	if h == (CharacterHeritage{}) {
		return nil
	}
	firstAncestryID := strings.TrimSpace(h.FirstFeatureAncestryID)
	secondAncestryID := strings.TrimSpace(h.SecondFeatureAncestryID)
	ancestryLabel := strings.TrimSpace(h.AncestryLabel)
	if strings.TrimSpace(h.FirstFeatureAncestryID) == "" || strings.TrimSpace(h.FirstFeatureID) == "" {
		return fmt.Errorf("validate daggerheart character profile: heritage first feature is required")
	}
	if strings.TrimSpace(h.SecondFeatureAncestryID) == "" || strings.TrimSpace(h.SecondFeatureID) == "" {
		return fmt.Errorf("validate daggerheart character profile: heritage second feature is required")
	}
	if strings.TrimSpace(h.CommunityID) == "" {
		return fmt.Errorf("validate daggerheart character profile: heritage community is required")
	}
	if firstAncestryID == secondAncestryID && ancestryLabel != "" {
		return fmt.Errorf("validate daggerheart character profile: ancestry label is only allowed for mixed ancestry")
	}
	return nil
}

// CharacterCompanionExperience captures one catalog-backed companion
// experience selection.
type CharacterCompanionExperience struct {
	ExperienceID string `json:"experience_id"`
	Modifier     int    `json:"modifier"`
}

// CharacterCompanionSheet captures the static companion sheet selected during
// Daggerheart character creation.
type CharacterCompanionSheet struct {
	AnimalKind        string                         `json:"animal_kind,omitempty"`
	Name              string                         `json:"name,omitempty"`
	Evasion           int                            `json:"evasion"`
	Experiences       []CharacterCompanionExperience `json:"experiences,omitempty"`
	AttackDescription string                         `json:"attack_description,omitempty"`
	AttackRange       string                         `json:"attack_range,omitempty"`
	DamageDieSides    int                            `json:"damage_die_sides"`
	DamageType        string                         `json:"damage_type,omitempty"`
}

// CharacterSubclassTrack captures one unlocked primary or multiclass subclass
// progression track.
type CharacterSubclassTrack struct {
	Origin     string `json:"origin,omitempty"`
	ClassID    string `json:"class_id,omitempty"`
	SubclassID string `json:"subclass_id,omitempty"`
	Rank       string `json:"rank,omitempty"`
	DomainID   string `json:"domain_id,omitempty"`
}

// Validate checks that the subclass track is internally valid.
func (t CharacterSubclassTrack) Validate() error {
	switch strings.TrimSpace(t.Origin) {
	case SubclassTrackOriginPrimary, SubclassTrackOriginMulticlass:
	default:
		return fmt.Errorf("validate daggerheart character profile: subclass track origin is invalid")
	}
	if strings.TrimSpace(t.ClassID) == "" {
		return fmt.Errorf("validate daggerheart character profile: subclass track class is required")
	}
	if strings.TrimSpace(t.SubclassID) == "" {
		return fmt.Errorf("validate daggerheart character profile: subclass track subclass is required")
	}
	switch strings.TrimSpace(t.Rank) {
	case SubclassTrackRankFoundation, SubclassTrackRankSpecialization, SubclassTrackRankMastery:
	default:
		return fmt.Errorf("validate daggerheart character profile: subclass track rank is invalid")
	}
	if strings.TrimSpace(t.Origin) == SubclassTrackOriginMulticlass && strings.TrimSpace(t.DomainID) == "" {
		return fmt.Errorf("validate daggerheart character profile: multiclass subclass track domain is required")
	}
	return nil
}

// Validate checks that the static companion sheet is internally valid.
func (c CharacterCompanionSheet) Validate() error {
	if strings.TrimSpace(c.AnimalKind) == "" {
		return fmt.Errorf("validate daggerheart character profile: companion animal kind is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("validate daggerheart character profile: companion name is required")
	}
	if len(c.Experiences) != 2 {
		return fmt.Errorf("validate daggerheart character profile: companion requires exactly two experiences")
	}
	for _, exp := range c.Experiences {
		if strings.TrimSpace(exp.ExperienceID) == "" {
			return fmt.Errorf("validate daggerheart character profile: companion experience id is required")
		}
		if exp.Modifier != CompanionSheetExperienceModifier {
			return fmt.Errorf("validate daggerheart character profile: companion experience modifier must be %d", CompanionSheetExperienceModifier)
		}
	}
	if strings.TrimSpace(c.AttackDescription) == "" {
		return fmt.Errorf("validate daggerheart character profile: companion attack description is required")
	}
	if c.Evasion != CompanionSheetDefaultEvasion {
		return fmt.Errorf("validate daggerheart character profile: companion evasion must be %d", CompanionSheetDefaultEvasion)
	}
	if c.AttackRange != CompanionSheetDefaultAttackRange {
		return fmt.Errorf("validate daggerheart character profile: companion attack range must be %q", CompanionSheetDefaultAttackRange)
	}
	if c.DamageDieSides != CompanionSheetDefaultDamageDieSides {
		return fmt.Errorf("validate daggerheart character profile: companion damage die must be d%d", CompanionSheetDefaultDamageDieSides)
	}
	switch strings.TrimSpace(c.DamageType) {
	case CompanionDamageTypePhysical, CompanionDamageTypeMagic:
		return nil
	default:
		return fmt.Errorf("validate daggerheart character profile: companion damage type must be physical or magic")
	}
}

// CharacterProfile is the authoritative Daggerheart character profile contract
// used by typed system-owned profile commands/events and aggregate state.
type CharacterProfile struct {
	Level                        int                          `json:"level"`
	HpMax                        int                          `json:"hp_max"`
	StressMax                    int                          `json:"stress_max"`
	Evasion                      int                          `json:"evasion"`
	MajorThreshold               int                          `json:"major_threshold"`
	SevereThreshold              int                          `json:"severe_threshold"`
	Proficiency                  int                          `json:"proficiency"`
	ArmorScore                   int                          `json:"armor_score"`
	ArmorMax                     int                          `json:"armor_max"`
	Experiences                  []CharacterProfileExperience `json:"experiences,omitempty"`
	Agility                      int                          `json:"agility"`
	Strength                     int                          `json:"strength"`
	Finesse                      int                          `json:"finesse"`
	Instinct                     int                          `json:"instinct"`
	Presence                     int                          `json:"presence"`
	Knowledge                    int                          `json:"knowledge"`
	ClassID                      string                       `json:"class_id,omitempty"`
	SubclassID                   string                       `json:"subclass_id,omitempty"`
	SubclassTracks               []CharacterSubclassTrack     `json:"subclass_tracks,omitempty"`
	SubclassCreationRequirements []string                     `json:"subclass_creation_requirements,omitempty"`
	Heritage                     CharacterHeritage            `json:"heritage,omitempty"`
	CompanionSheet               *CharacterCompanionSheet     `json:"companion_sheet,omitempty"`
	EquippedArmorID              string                       `json:"equipped_armor_id,omitempty"`
	SpellcastRollBonus           int                          `json:"spellcast_roll_bonus,omitempty"`
	TraitsAssigned               bool                         `json:"traits_assigned"`
	DetailsRecorded              bool                         `json:"details_recorded"`
	StartingWeaponIDs            []string                     `json:"starting_weapon_ids,omitempty"`
	StartingArmorID              string                       `json:"starting_armor_id,omitempty"`
	StartingPotionItemID         string                       `json:"starting_potion_item_id,omitempty"`
	Background                   string                       `json:"background,omitempty"`
	Description                  string                       `json:"description,omitempty"`
	DomainCardIDs                []string                     `json:"domain_card_ids,omitempty"`
	Connections                  string                       `json:"connections,omitempty"`
	GoldHandfuls                 int                          `json:"gold_handfuls,omitempty"`
	GoldBags                     int                          `json:"gold_bags,omitempty"`
	GoldChests                   int                          `json:"gold_chests,omitempty"`
}

// MutationSource records the reason a profile mutation happened so callers
// can attribute changes to domain cards, features, or GM adjustments.
type MutationSource struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
}

// CharacterProfileReplacePayload captures the payload for
// sys.daggerheart.character_profile.replace commands.
type CharacterProfileReplacePayload struct {
	CharacterID    ids.CharacterID  `json:"character_id"`
	Profile        CharacterProfile `json:"profile"`
	MutationSource *MutationSource  `json:"mutation_source,omitempty"`
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
	if len(normalized.SubclassTracks) == 0 &&
		strings.TrimSpace(normalized.ClassID) != "" &&
		strings.TrimSpace(normalized.SubclassID) != "" {
		normalized.SubclassTracks = EnsurePrimarySubclassTrack(nil, normalized.ClassID, normalized.SubclassID)
	}
	if normalized.CompanionSheet != nil {
		companion := *normalized.CompanionSheet
		companion.Evasion = CompanionSheetDefaultEvasion
		companion.AttackRange = CompanionSheetDefaultAttackRange
		companion.DamageDieSides = CompanionSheetDefaultDamageDieSides
		if len(companion.Experiences) > 0 {
			experiences := make([]CharacterCompanionExperience, 0, len(companion.Experiences))
			for _, exp := range companion.Experiences {
				experiences = append(experiences, CharacterCompanionExperience{
					ExperienceID: strings.TrimSpace(exp.ExperienceID),
					Modifier:     CompanionSheetExperienceModifier,
				})
			}
			companion.Experiences = experiences
		}
		normalized.CompanionSheet = &companion
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
	if err := normalized.Heritage.Validate(); err != nil {
		return err
	}
	if err := ValidateSubclassCreationRequirements(normalized.SubclassCreationRequirements); err != nil {
		return err
	}
	if err := ValidateSubclassTracks(normalized.ClassID, normalized.SubclassID, normalized.SubclassTracks); err != nil {
		return err
	}
	if RequiresCompanionSheet(normalized.SubclassCreationRequirements) && normalized.CompanionSheet == nil {
		return fmt.Errorf("validate daggerheart character profile: companion sheet is required")
	}
	if normalized.CompanionSheet != nil {
		if err := normalized.CompanionSheet.Validate(); err != nil {
			return err
		}
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

	var requirements []projectionstore.DaggerheartSubclassCreationRequirement
	if len(normalized.SubclassCreationRequirements) > 0 {
		requirements = make([]projectionstore.DaggerheartSubclassCreationRequirement, 0, len(normalized.SubclassCreationRequirements))
		for _, requirement := range normalized.SubclassCreationRequirements {
			requirements = append(requirements, projectionstore.DaggerheartSubclassCreationRequirement(strings.TrimSpace(requirement)))
		}
	}
	var subclassTracks []projectionstore.DaggerheartSubclassTrack
	if len(normalized.SubclassTracks) > 0 {
		subclassTracks = make([]projectionstore.DaggerheartSubclassTrack, 0, len(normalized.SubclassTracks))
		for _, track := range normalized.SubclassTracks {
			subclassTracks = append(subclassTracks, projectionstore.DaggerheartSubclassTrack{
				Origin:     projectionstore.DaggerheartSubclassTrackOrigin(strings.TrimSpace(track.Origin)),
				ClassID:    strings.TrimSpace(track.ClassID),
				SubclassID: strings.TrimSpace(track.SubclassID),
				Rank:       projectionstore.DaggerheartSubclassTrackRank(strings.TrimSpace(track.Rank)),
				DomainID:   strings.TrimSpace(track.DomainID),
			})
		}
	}

	var companion *projectionstore.DaggerheartCompanionSheet
	if normalized.CompanionSheet != nil {
		companionExperiences := make([]projectionstore.DaggerheartCompanionExperience, 0, len(normalized.CompanionSheet.Experiences))
		for _, exp := range normalized.CompanionSheet.Experiences {
			companionExperiences = append(companionExperiences, projectionstore.DaggerheartCompanionExperience{
				ExperienceID: exp.ExperienceID,
				Modifier:     exp.Modifier,
			})
		}
		companion = &projectionstore.DaggerheartCompanionSheet{
			AnimalKind:        normalized.CompanionSheet.AnimalKind,
			Name:              normalized.CompanionSheet.Name,
			Evasion:           normalized.CompanionSheet.Evasion,
			Experiences:       companionExperiences,
			AttackDescription: normalized.CompanionSheet.AttackDescription,
			AttackRange:       normalized.CompanionSheet.AttackRange,
			DamageDieSides:    normalized.CompanionSheet.DamageDieSides,
			DamageType:        normalized.CompanionSheet.DamageType,
		}
	}

	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:                   strings.TrimSpace(campaignID),
		CharacterID:                  strings.TrimSpace(characterID),
		Level:                        normalized.Level,
		HpMax:                        normalized.HpMax,
		StressMax:                    normalized.StressMax,
		Evasion:                      normalized.Evasion,
		MajorThreshold:               normalized.MajorThreshold,
		SevereThreshold:              normalized.SevereThreshold,
		Proficiency:                  normalized.Proficiency,
		ArmorScore:                   normalized.ArmorScore,
		ArmorMax:                     normalized.ArmorMax,
		Experiences:                  experiences,
		ClassID:                      normalized.ClassID,
		SubclassID:                   normalized.SubclassID,
		SubclassTracks:               subclassTracks,
		SubclassCreationRequirements: requirements,
		Heritage: projectionstore.DaggerheartHeritageSelection{
			AncestryLabel:           normalized.Heritage.AncestryLabel,
			FirstFeatureAncestryID:  normalized.Heritage.FirstFeatureAncestryID,
			FirstFeatureID:          normalized.Heritage.FirstFeatureID,
			SecondFeatureAncestryID: normalized.Heritage.SecondFeatureAncestryID,
			SecondFeatureID:         normalized.Heritage.SecondFeatureID,
			CommunityID:             normalized.Heritage.CommunityID,
		},
		CompanionSheet:       companion,
		EquippedArmorID:      normalized.EquippedArmorID,
		SpellcastRollBonus:   normalized.SpellcastRollBonus,
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

	var requirements []string
	if len(profile.SubclassCreationRequirements) > 0 {
		requirements = make([]string, 0, len(profile.SubclassCreationRequirements))
		for _, requirement := range profile.SubclassCreationRequirements {
			requirements = append(requirements, string(requirement))
		}
	}
	var subclassTracks []CharacterSubclassTrack
	if len(profile.SubclassTracks) > 0 {
		subclassTracks = make([]CharacterSubclassTrack, 0, len(profile.SubclassTracks))
		for _, track := range profile.SubclassTracks {
			subclassTracks = append(subclassTracks, CharacterSubclassTrack{
				Origin:     strings.TrimSpace(string(track.Origin)),
				ClassID:    track.ClassID,
				SubclassID: track.SubclassID,
				Rank:       strings.TrimSpace(string(track.Rank)),
				DomainID:   track.DomainID,
			})
		}
	}

	var companion *CharacterCompanionSheet
	if profile.CompanionSheet != nil {
		companionExperiences := make([]CharacterCompanionExperience, 0, len(profile.CompanionSheet.Experiences))
		for _, exp := range profile.CompanionSheet.Experiences {
			companionExperiences = append(companionExperiences, CharacterCompanionExperience{
				ExperienceID: exp.ExperienceID,
				Modifier:     exp.Modifier,
			})
		}
		companion = &CharacterCompanionSheet{
			AnimalKind:        profile.CompanionSheet.AnimalKind,
			Name:              profile.CompanionSheet.Name,
			Evasion:           profile.CompanionSheet.Evasion,
			Experiences:       companionExperiences,
			AttackDescription: profile.CompanionSheet.AttackDescription,
			AttackRange:       profile.CompanionSheet.AttackRange,
			DamageDieSides:    profile.CompanionSheet.DamageDieSides,
			DamageType:        profile.CompanionSheet.DamageType,
		}
	}

	return CharacterProfile{
		Level:                        profile.Level,
		HpMax:                        profile.HpMax,
		StressMax:                    profile.StressMax,
		Evasion:                      profile.Evasion,
		MajorThreshold:               profile.MajorThreshold,
		SevereThreshold:              profile.SevereThreshold,
		Proficiency:                  profile.Proficiency,
		ArmorScore:                   profile.ArmorScore,
		ArmorMax:                     profile.ArmorMax,
		Experiences:                  experiences,
		Agility:                      profile.Agility,
		Strength:                     profile.Strength,
		Finesse:                      profile.Finesse,
		Instinct:                     profile.Instinct,
		Presence:                     profile.Presence,
		Knowledge:                    profile.Knowledge,
		ClassID:                      profile.ClassID,
		SubclassID:                   profile.SubclassID,
		SubclassTracks:               subclassTracks,
		SubclassCreationRequirements: requirements,
		Heritage: CharacterHeritage{
			AncestryLabel:           profile.Heritage.AncestryLabel,
			FirstFeatureAncestryID:  profile.Heritage.FirstFeatureAncestryID,
			FirstFeatureID:          profile.Heritage.FirstFeatureID,
			SecondFeatureAncestryID: profile.Heritage.SecondFeatureAncestryID,
			SecondFeatureID:         profile.Heritage.SecondFeatureID,
			CommunityID:             profile.Heritage.CommunityID,
		},
		CompanionSheet:       companion,
		EquippedArmorID:      profile.EquippedArmorID,
		SpellcastRollBonus:   profile.SpellcastRollBonus,
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
		ClassID:                      normalized.ClassID,
		SubclassID:                   normalized.SubclassID,
		SubclassTracks:               append([]CharacterSubclassTrack(nil), normalized.SubclassTracks...),
		SubclassCreationRequirements: append([]string(nil), normalized.SubclassCreationRequirements...),
		Heritage:                     normalized.Heritage,
		CompanionSheet:               normalized.CompanionSheet,
		EquippedArmorID:              normalized.EquippedArmorID,
		SpellcastRollBonus:           normalized.SpellcastRollBonus,
		TraitsAssigned:               normalized.TraitsAssigned,
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

// ValidateSubclassCreationRequirements validates creation requirements.
func ValidateSubclassCreationRequirements(requirements []string) error {
	for _, requirement := range requirements {
		switch strings.TrimSpace(requirement) {
		case "", SubclassCreationRequirementCompanionSheet:
		default:
			return fmt.Errorf("validate daggerheart character profile: unsupported subclass creation requirement %q", requirement)
		}
	}
	return nil
}

// RequiresCompanionSheet reports whether the creation requirements include a companion sheet.
func RequiresCompanionSheet(requirements []string) bool {
	for _, requirement := range requirements {
		if strings.TrimSpace(requirement) == SubclassCreationRequirementCompanionSheet {
			return true
		}
	}
	return false
}

// ValidateSubclassTracks validates the subclass track list consistency.
func ValidateSubclassTracks(classID, subclassID string, tracks []CharacterSubclassTrack) error {
	if len(tracks) == 0 {
		return nil
	}
	var sawPrimary bool
	for _, track := range tracks {
		if err := track.Validate(); err != nil {
			return err
		}
		switch strings.TrimSpace(track.Origin) {
		case SubclassTrackOriginPrimary:
			if sawPrimary {
				return fmt.Errorf("validate daggerheart character profile: duplicate primary subclass track")
			}
			sawPrimary = true
			if strings.TrimSpace(track.ClassID) != strings.TrimSpace(classID) {
				return fmt.Errorf("validate daggerheart character profile: primary subclass track must match class_id")
			}
			if strings.TrimSpace(track.SubclassID) != strings.TrimSpace(subclassID) {
				return fmt.Errorf("validate daggerheart character profile: primary subclass track must match subclass_id")
			}
		}
	}
	if strings.TrimSpace(classID) != "" && strings.TrimSpace(subclassID) != "" && !sawPrimary {
		return fmt.Errorf("validate daggerheart character profile: primary subclass track is required")
	}
	return nil
}

// CreationProfile captures Daggerheart-specific character-creation choices.
type CreationProfile struct {
	ClassID                      string
	SubclassID                   string
	SubclassTracks               []CharacterSubclassTrack
	SubclassCreationRequirements []string
	Heritage                     CharacterHeritage
	CompanionSheet               *CharacterCompanionSheet
	EquippedArmorID              string
	SpellcastRollBonus           int
	TraitsAssigned               bool
	Traits                       daggerheartprofile.Traits
	DetailsRecorded              bool
	Level                        int
	HpMax                        int
	StressMax                    int
	Evasion                      int
	StartingWeaponIDs            []string
	StartingArmorID              string
	StartingPotionItemID         string
	Background                   string
	Description                  string
	Experiences                  []daggerheartprofile.Experience
	DomainCardIDs                []string
	Connections                  string
}
