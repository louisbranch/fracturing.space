package projection

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ValidateAdversaryStats validates read-model adversary stat ranges.
func ValidateAdversaryStats(hp, hpMax, stress, stressMax, evasion, major, severe, armor int) error {
	if hpMax <= 0 {
		return fmt.Errorf("hp_max must be positive")
	}
	if hp < 0 || hp > hpMax {
		return fmt.Errorf("hp must be in range 0..%d", hpMax)
	}
	if stressMax < 0 {
		return fmt.Errorf("stress_max must be non-negative")
	}
	if stress < 0 || stress > stressMax {
		return fmt.Errorf("stress must be in range 0..%d", stressMax)
	}
	if evasion < 0 {
		return fmt.Errorf("evasion must be non-negative")
	}
	if major < 0 || severe < 0 {
		return fmt.Errorf("thresholds must be non-negative")
	}
	if severe < major {
		return fmt.Errorf("severe_threshold must be >= major_threshold")
	}
	if armor < 0 {
		return fmt.Errorf("armor must be non-negative")
	}
	return nil
}

// ApplyAdversaryConditionPatch replaces adversary conditions.
func ApplyAdversaryConditionPatch(adversary storage.DaggerheartAdversary, conditions []string) storage.DaggerheartAdversary {
	next := adversary
	next.Conditions = append([]string(nil), conditions...)
	return next
}

// ApplyAdversaryDamagePatch merges optional hp/armor updates and validates the
// resulting state before persistence.
func ApplyAdversaryDamagePatch(
	adversary storage.DaggerheartAdversary,
	hpAfter *int,
	armorAfter *int,
) (storage.DaggerheartAdversary, error) {
	next := adversary
	if hpAfter != nil {
		next.HP = *hpAfter
	}
	if armorAfter != nil {
		next.Armor = *armorAfter
	}
	if err := ValidateAdversaryStats(
		next.HP,
		next.HPMax,
		next.Stress,
		next.StressMax,
		next.Evasion,
		next.Major,
		next.Severe,
		next.Armor,
	); err != nil {
		return storage.DaggerheartAdversary{}, err
	}
	return next, nil
}
