package profile

import (
	"fmt"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/mechanics"
)

const (
	PCLevelDefault    = 1
	PCHpMax           = 6
	PCStressMax       = 6
	PCEvasion         = 10
	PCMajorThreshold  = 8
	PCSevereThreshold = 12
	PCProficiency     = 1
	PCArmorScore      = 0
	PCArmorMax        = 0

	AdversaryDefaultEvasion = PCEvasion
	AdversaryDefaultMajor   = PCMajorThreshold
	AdversaryDefaultSevere  = PCSevereThreshold

	NPCLevelDefault    = 1
	NPCHpMax           = 3
	NPCStressMax       = 3
	NPCEvasion         = 8
	NPCMajorThreshold  = 6
	NPCSevereThreshold = 10
	NPCProficiency     = 0
	NPCArmorScore      = 0
	NPCArmorMax        = 0

	TraitDefault = 0
	TraitMin     = -2
	TraitMax     = 4

	HPMaxCap     = mechanics.HPMaxCap
	StressMaxCap = mechanics.StressMaxCap
	ArmorMaxCap  = mechanics.ArmorMaxCap
)

var (
	ErrInvalidLevel       = apperrors.New(apperrors.CodeDaggerheartInvalidLevel, "level must be in range 1..10")
	ErrInvalidTraitValue  = apperrors.New(apperrors.CodeDaggerheartInvalidTraitValue, "trait values must be in range -2..+4")
	ErrInvalidStressMax   = apperrors.New(apperrors.CodeDaggerheartInvalidStressMax, "stress_max must be in range 0..12")
	ErrInvalidHpMax       = apperrors.New(apperrors.CodeDaggerheartInvalidHpMax, "hp_max must be in range 1..12")
	ErrInvalidEvasion     = apperrors.New(apperrors.CodeDaggerheartInvalidEvasion, "evasion must be non-negative")
	ErrInvalidThresholds  = apperrors.New(apperrors.CodeDaggerheartInvalidThresholds, "severe_threshold must be >= major_threshold >= 0")
	ErrInvalidProficiency = apperrors.New(apperrors.CodeDaggerheartInvalidProficiency, "proficiency must be non-negative")
	ErrInvalidArmorMax    = apperrors.New(apperrors.CodeDaggerheartInvalidArmorMax, "armor_max must be in range 0..12")
	ErrInvalidArmorScore  = apperrors.New(apperrors.CodeDaggerheartInvalidArmorScore, "armor_score must be non-negative")
	ErrInvalidExperience  = apperrors.New(apperrors.CodeDaggerheartInvalidExperience, "experience name must be set")
)

type Experience struct {
	Name     string
	Modifier int
}

type Traits struct {
	Agility   int
	Strength  int
	Finesse   int
	Instinct  int
	Presence  int
	Knowledge int
}

func DefaultTraits() Traits {
	return Traits{}
}

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

func ValidateTraits(traits Traits) error {
	values := map[string]int{
		"agility":   traits.Agility,
		"strength":  traits.Strength,
		"finesse":   traits.Finesse,
		"instinct":  traits.Instinct,
		"presence":  traits.Presence,
		"knowledge": traits.Knowledge,
	}
	for name, value := range values {
		if err := ValidateTrait(name, value); err != nil {
			return err
		}
	}
	return nil
}

type Defaults struct {
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

func GetDefaults(kind string) Defaults {
	switch kind {
	case "NPC":
		level := NPCLevelDefault
		major, severe := DeriveThresholds(level, NPCArmorScore, NPCMajorThreshold, NPCSevereThreshold)
		return Defaults{
			Level:           level,
			HpMax:           NPCHpMax,
			StressMax:       NPCStressMax,
			Evasion:         NPCEvasion,
			MajorThreshold:  major,
			SevereThreshold: severe,
			Proficiency:     NPCProficiency,
			ArmorScore:      NPCArmorScore,
			ArmorMax:        NPCArmorMax,
			Traits:          DefaultTraits(),
		}
	default:
		level := PCLevelDefault
		major, severe := DeriveThresholds(level, PCArmorScore, PCMajorThreshold, PCSevereThreshold)
		return Defaults{
			Level:           level,
			HpMax:           PCHpMax,
			StressMax:       PCStressMax,
			Evasion:         PCEvasion,
			MajorThreshold:  major,
			SevereThreshold: severe,
			Proficiency:     PCProficiency,
			ArmorScore:      PCArmorScore,
			ArmorMax:        PCArmorMax,
			Traits:          DefaultTraits(),
		}
	}
}

func UnarmoredThresholds(level int) (majorThreshold int, severeThreshold int) {
	return level, level * 2
}

func DeriveThresholds(level, armorScore, majorThreshold, severeThreshold int) (int, int) {
	if armorScore == 0 {
		return UnarmoredThresholds(level)
	}
	return majorThreshold, severeThreshold
}

func ValidateLevel(level int) error {
	if level < 1 || level > 10 {
		return ErrInvalidLevel
	}
	return nil
}

func Validate(
	level,
	hpMax,
	stressMax,
	evasion,
	majorThreshold,
	severeThreshold,
	proficiency,
	armorScore,
	armorMax int,
	traits Traits,
	experiences []Experience,
) error {
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
