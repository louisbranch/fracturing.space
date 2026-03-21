package rules

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"

// DamageApplyInput captures transport-agnostic inputs for applying damage.
type DamageApplyInput struct {
	Amount       int
	Types        DamageTypes
	Resistance   ResistanceProfile
	Direct       bool
	AllowMassive bool
}

// DamageTarget captures current state and thresholds for one damage target.
type DamageTarget struct {
	HP              int
	Stress          int
	Armor           int
	MajorThreshold  int
	SevereThreshold int
	ArmorRules      ArmorDamageRules
}

// ResolveDamageApplication computes and applies damage for a target using
// Daggerheart damage rules. It returns the resulting application and whether
// any mitigation (resistance, immunity, or armor spend) occurred.
func ResolveDamageApplication(target DamageTarget, input DamageApplyInput) (DamageApplication, bool, error) {
	adjusted := ApplyResistance(input.Amount, input.Types, input.Resistance)
	mitigated := adjusted < input.Amount
	options := DamageOptions{EnableMassiveDamage: input.AllowMassive}
	if target.ArmorRules.WardedMagicReduction && input.Types.Magic && !input.Types.Physical {
		adjusted -= target.ArmorRules.WardedReductionAmount
		if adjusted < 0 {
			adjusted = 0
		}
		if adjusted < input.Amount {
			mitigated = true
		}
	}

	result, err := EvaluateDamage(adjusted, target.MajorThreshold, target.SevereThreshold, options)
	if err != nil {
		return DamageApplication{}, mitigated, err
	}

	if input.Direct {
		application, applyErr := ApplyDamage(target.HP, adjusted, target.MajorThreshold, target.SevereThreshold, options)
		return application, mitigated, applyErr
	}

	if !ArmorCanMitigate(target.ArmorRules, input.Types) {
		application, applyErr := ApplyDamage(target.HP, adjusted, target.MajorThreshold, target.SevereThreshold, options)
		if applyErr != nil {
			return DamageApplication{}, mitigated, applyErr
		}
		application.StressBefore = target.Stress
		application.StressAfter = target.Stress
		return application, mitigated, nil
	}

	application := ApplyDamageWithArmor(target.HP, target.Stress, target.Armor, result, target.ArmorRules)
	if application.ArmorSpent > 0 || application.StressAfter != application.StressBefore {
		mitigated = true
	}
	return application, mitigated, nil
}

// ArmorCanMitigate reports whether the given armor rules allow mitigation for
// the specified damage types.
func ArmorCanMitigate(rules ArmorDamageRules, types DamageTypes) bool {
	switch rules.MitigationMode {
	case string(contentstore.DaggerheartArmorMitigationModePhysicalOnly):
		return types.Physical && !types.Magic
	case string(contentstore.DaggerheartArmorMitigationModeMagicOnly):
		return types.Magic && !types.Physical
	default:
		return true
	}
}
