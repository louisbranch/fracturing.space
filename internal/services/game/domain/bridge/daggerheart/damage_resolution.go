package daggerheart

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
	Armor           int
	MajorThreshold  int
	SevereThreshold int
}

// ResolveDamageApplication computes and applies damage for a target using
// Daggerheart damage rules. It returns the resulting application and whether
// any mitigation (resistance, immunity, or armor spend) occurred.
func ResolveDamageApplication(target DamageTarget, input DamageApplyInput) (DamageApplication, bool, error) {
	adjusted := ApplyResistance(input.Amount, input.Types, input.Resistance)
	mitigated := adjusted < input.Amount
	options := DamageOptions{EnableMassiveDamage: input.AllowMassive}

	result, err := EvaluateDamage(adjusted, target.MajorThreshold, target.SevereThreshold, options)
	if err != nil {
		return DamageApplication{}, mitigated, err
	}

	if input.Direct {
		application, applyErr := ApplyDamage(target.HP, adjusted, target.MajorThreshold, target.SevereThreshold, options)
		return application, mitigated, applyErr
	}

	application := ApplyDamageWithArmor(target.HP, target.Armor, result)
	if application.ArmorSpent > 0 {
		mitigated = true
	}
	return application, mitigated, nil
}
