package daggerheart

import (
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

// Profile defaults for Daggerheart characters.
const (
	// PC defaults
	PCLevelDefault    = 1
	PCHpMax           = 6
	PCStressMax       = 6
	PCEvasion         = 10
	PCMajorThreshold  = 8
	PCSevereThreshold = 12
	PCProficiency     = 1
	PCArmorScore      = 0
	PCArmorMax        = 0

	// NPC defaults
	NPCLevelDefault    = 1
	NPCHpMax           = 3
	NPCStressMax       = 3
	NPCEvasion         = 8
	NPCMajorThreshold  = 6
	NPCSevereThreshold = 10
	NPCProficiency     = 0
	NPCArmorScore      = 0
	NPCArmorMax        = 0

	// Trait defaults (all traits start at 0)
	TraitDefault = 0
	TraitMin     = -2
	TraitMax     = 4

	ArmorMaxCap = 12
)

var (
	// ErrInvalidLevel indicates level is out of range.
	ErrInvalidLevel = apperrors.New(apperrors.CodeDaggerheartInvalidLevel, "level must be in range 1..10")
	// ErrInvalidTraitValue indicates a trait value is outside the valid range.
	ErrInvalidTraitValue = apperrors.New(apperrors.CodeDaggerheartInvalidTraitValue, "trait values must be in range -2..+4")
	// ErrInvalidStressMax indicates stress_max is out of range.
	ErrInvalidStressMax = apperrors.New(apperrors.CodeDaggerheartInvalidStressMax, "stress_max must be in range 0..12")
	// ErrInvalidHpMax indicates hp_max is out of range.
	ErrInvalidHpMax = apperrors.New(apperrors.CodeDaggerheartInvalidHpMax, "hp_max must be in range 1..12")
	// ErrInvalidEvasion indicates evasion is negative.
	ErrInvalidEvasion = apperrors.New(apperrors.CodeDaggerheartInvalidEvasion, "evasion must be non-negative")
	// ErrInvalidThresholds indicates threshold ordering is invalid.
	ErrInvalidThresholds = apperrors.New(apperrors.CodeDaggerheartInvalidThresholds, "severe_threshold must be >= major_threshold >= 0")
	// ErrInvalidProficiency indicates proficiency is negative.
	ErrInvalidProficiency = apperrors.New(apperrors.CodeDaggerheartInvalidProficiency, "proficiency must be non-negative")
	// ErrInvalidArmorMax indicates armor_max is out of range.
	ErrInvalidArmorMax = apperrors.New(apperrors.CodeDaggerheartInvalidArmorMax, "armor_max must be in range 0..12")
	// ErrInvalidArmorScore indicates armor_score is negative.
	ErrInvalidArmorScore = apperrors.New(apperrors.CodeDaggerheartInvalidArmorScore, "armor_score must be non-negative")
	// ErrInvalidExperience indicates an experience entry is invalid.
	ErrInvalidExperience = apperrors.New(apperrors.CodeDaggerheartInvalidExperience, "experience name must be set")
)

// Experience represents a named experience modifier.
type Experience struct {
	Name     string
	Modifier int
}

// Traits represents the six Daggerheart character traits.
type Traits struct {
	Agility   int
	Strength  int
	Finesse   int
	Instinct  int
	Presence  int
	Knowledge int
}

// DefaultTraits returns the default trait values for new characters.
func DefaultTraits() Traits {
	return Traits{
		Agility:   TraitDefault,
		Strength:  TraitDefault,
		Finesse:   TraitDefault,
		Instinct:  TraitDefault,
		Presence:  TraitDefault,
		Knowledge: TraitDefault,
	}
}

// ValidateTrait validates a single trait value is within range.
func ValidateTrait(name string, value int) error {
	if value < TraitMin || value > TraitMax {
		return apperrors.WithMetadata(
			apperrors.CodeDaggerheartInvalidTraitValue,
			fmt.Sprintf("trait %q has value %d, must be in range %d..%d", name, value, TraitMin, TraitMax),
			map[string]string{"Trait": name, "Value": fmt.Sprintf("%d", value)},
		)
	}
	return nil
}

// ValidateTraits validates all trait values.
func ValidateTraits(t Traits) error {
	traits := map[string]int{
		"agility":   t.Agility,
		"strength":  t.Strength,
		"finesse":   t.Finesse,
		"instinct":  t.Instinct,
		"presence":  t.Presence,
		"knowledge": t.Knowledge,
	}
	for name, value := range traits {
		if err := ValidateTrait(name, value); err != nil {
			return err
		}
	}
	return nil
}

// ProfileDefaults contains default profile values for a character kind.
type ProfileDefaults struct {
	Level           int
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
	Proficiency     int
	ArmorScore      int
	ArmorMax        int
	Traits          Traits
	Experiences     []Experience
}

// GetProfileDefaults returns profile defaults for a character kind.
// kind should be "PC" or "NPC".
func GetProfileDefaults(kind string) ProfileDefaults {
	switch kind {
	case "NPC":
		return ProfileDefaults{
			Level:           NPCLevelDefault,
			HpMax:           NPCHpMax,
			StressMax:       NPCStressMax,
			Evasion:         NPCEvasion,
			MajorThreshold:  NPCMajorThreshold,
			SevereThreshold: NPCSevereThreshold,
			Proficiency:     NPCProficiency,
			ArmorScore:      NPCArmorScore,
			ArmorMax:        NPCArmorMax,
			Traits:          DefaultTraits(),
			Experiences:     nil,
		}
	default: // PC
		return ProfileDefaults{
			Level:           PCLevelDefault,
			HpMax:           PCHpMax,
			StressMax:       PCStressMax,
			Evasion:         PCEvasion,
			MajorThreshold:  PCMajorThreshold,
			SevereThreshold: PCSevereThreshold,
			Proficiency:     PCProficiency,
			ArmorScore:      PCArmorScore,
			ArmorMax:        PCArmorMax,
			Traits:          DefaultTraits(),
			Experiences:     nil,
		}
	}
}

// ValidateLevel validates level is within 1..10.
func ValidateLevel(level int) error {
	if level < 1 || level > 10 {
		return ErrInvalidLevel
	}
	return nil
}

// ValidateProfile validates Daggerheart-specific profile fields.
func ValidateProfile(level, hpMax, stressMax, evasion, majorThreshold, severeThreshold, proficiency, armorScore, armorMax int, traits Traits, experiences []Experience) error {
	if err := ValidateLevel(level); err != nil {
		return err
	}
	if hpMax < 1 || hpMax > HPMaxCap {
		return ErrInvalidHpMax
	}
	if stressMax < 0 || stressMax > StressMaxCap {
		return ErrInvalidStressMax
	}
	if evasion < 0 {
		return ErrInvalidEvasion
	}
	if majorThreshold < 0 || severeThreshold < majorThreshold {
		return ErrInvalidThresholds
	}
	if proficiency < 0 {
		return ErrInvalidProficiency
	}
	if armorScore < 0 {
		return ErrInvalidArmorScore
	}
	if armorMax < 0 || armorMax > ArmorMaxCap {
		return ErrInvalidArmorMax
	}
	for _, experience := range experiences {
		if experience.Name == "" {
			return ErrInvalidExperience
		}
	}
	return ValidateTraits(traits)
}
