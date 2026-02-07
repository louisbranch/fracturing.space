package daggerheart

import (
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
)

// Profile defaults for Daggerheart characters.
const (
	// PC defaults
	PCHpMax           = 6
	PCStressMax       = 6
	PCEvasion         = 10
	PCMajorThreshold  = 8
	PCSevereThreshold = 12

	// NPC defaults
	NPCHpMax           = 3
	NPCStressMax       = 3
	NPCEvasion         = 8
	NPCMajorThreshold  = 6
	NPCSevereThreshold = 10

	// Trait defaults (all traits start at 0)
	TraitDefault = 0
	TraitMin     = -2
	TraitMax     = 4
)

var (
	// ErrInvalidTraitValue indicates a trait value is outside the valid range.
	ErrInvalidTraitValue = apperrors.New(apperrors.CodeDaggerheartInvalidTraitValue, "trait values must be in range -2..+4")
	// ErrInvalidStressMax indicates stress_max is negative.
	ErrInvalidStressMax = apperrors.New(apperrors.CodeDaggerheartInvalidStressMax, "stress_max must be non-negative")
	// ErrInvalidHpMax indicates hp_max is less than 1.
	ErrInvalidHpMax = apperrors.New(apperrors.CodeDaggerheartInvalidHpMax, "hp_max must be at least 1")
	// ErrInvalidEvasion indicates evasion is negative.
	ErrInvalidEvasion = apperrors.New(apperrors.CodeDaggerheartInvalidEvasion, "evasion must be non-negative")
	// ErrInvalidThresholds indicates threshold ordering is invalid.
	ErrInvalidThresholds = apperrors.New(apperrors.CodeDaggerheartInvalidThresholds, "severe_threshold must be >= major_threshold >= 0")
)

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
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
	Traits          Traits
}

// GetProfileDefaults returns profile defaults for a character kind.
// kind should be "PC" or "NPC".
func GetProfileDefaults(kind string) ProfileDefaults {
	switch kind {
	case "NPC":
		return ProfileDefaults{
			HpMax:           NPCHpMax,
			StressMax:       NPCStressMax,
			Evasion:         NPCEvasion,
			MajorThreshold:  NPCMajorThreshold,
			SevereThreshold: NPCSevereThreshold,
			Traits:          DefaultTraits(),
		}
	default: // PC
		return ProfileDefaults{
			HpMax:           PCHpMax,
			StressMax:       PCStressMax,
			Evasion:         PCEvasion,
			MajorThreshold:  PCMajorThreshold,
			SevereThreshold: PCSevereThreshold,
			Traits:          DefaultTraits(),
		}
	}
}

// ValidateProfile validates Daggerheart-specific profile fields.
func ValidateProfile(stressMax, evasion, majorThreshold, severeThreshold int, traits Traits) error {
	if stressMax < 0 {
		return ErrInvalidStressMax
	}
	if evasion < 0 {
		return ErrInvalidEvasion
	}
	if majorThreshold < 0 || severeThreshold < majorThreshold {
		return ErrInvalidThresholds
	}
	return ValidateTraits(traits)
}
