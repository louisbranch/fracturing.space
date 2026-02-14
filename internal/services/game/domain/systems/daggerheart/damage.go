package daggerheart

// DamageSeverity describes the severity tier of incoming damage.
type DamageSeverity int

const (
	DamageNone DamageSeverity = iota
	DamageMinor
	DamageMajor
	DamageSevere
	DamageMassive
)

// DamageType represents damage categories in Daggerheart.
type DamageType int

const (
	DamageTypePhysical DamageType = iota
	DamageTypeMagic
)

// DamageTypes flags the damage categories for an attack.
type DamageTypes struct {
	Physical bool
	Magic    bool
}

// ResistanceProfile describes resistance and immunity to damage types.
type ResistanceProfile struct {
	ResistPhysical bool
	ResistMagic    bool
	ImmunePhysical bool
	ImmuneMagic    bool
}

// DamageOptions configures optional damage rules.
type DamageOptions struct {
	// EnableMassiveDamage applies the optional massive damage rule.
	EnableMassiveDamage bool
}

// DamageResult describes the damage severity and HP marks to apply.
type DamageResult struct {
	Severity DamageSeverity
	Marks    int
}

// DamageApplication captures HP deltas alongside damage evaluation.
type DamageApplication struct {
	Result      DamageResult
	HPBefore    int
	HPAfter     int
	ArmorBefore int
	ArmorAfter  int
	ArmorSpent  int
}

// EvaluateDamage determines severity and HP marks from damage totals and thresholds.
// Thresholds are expected to already include any level adjustments.
func EvaluateDamage(amount, majorThreshold, severeThreshold int, opts DamageOptions) (DamageResult, error) {
	if majorThreshold < 0 || severeThreshold < majorThreshold {
		return DamageResult{}, ErrInvalidThresholds
	}
	if amount <= 0 {
		return DamageResult{Severity: DamageNone, Marks: 0}, nil
	}
	if opts.EnableMassiveDamage && amount >= severeThreshold*2 {
		return DamageResult{Severity: DamageMassive, Marks: 4}, nil
	}
	if amount >= severeThreshold {
		return DamageResult{Severity: DamageSevere, Marks: 3}, nil
	}
	if amount >= majorThreshold {
		return DamageResult{Severity: DamageMajor, Marks: 2}, nil
	}
	return DamageResult{Severity: DamageMinor, Marks: 1}, nil
}

// ApplyDamageMarks reduces current HP by the given number of marks.
func ApplyDamageMarks(currentHP, marks int) (before, after int) {
	before = currentHP
	if marks <= 0 {
		return before, before
	}
	if currentHP <= 0 {
		return before, 0
	}
	remaining := currentHP - marks
	if remaining < 0 {
		remaining = 0
	}
	return before, remaining
}

// ApplyDamage evaluates damage severity and applies resulting HP marks.
func ApplyDamage(currentHP, amount, majorThreshold, severeThreshold int, opts DamageOptions) (DamageApplication, error) {
	result, err := EvaluateDamage(amount, majorThreshold, severeThreshold, opts)
	if err != nil {
		return DamageApplication{}, err
	}
	before, after := ApplyDamageMarks(currentHP, result.Marks)
	return DamageApplication{
		Result:      result,
		HPBefore:    before,
		HPAfter:     after,
		ArmorBefore: 0,
		ArmorAfter:  0,
	}, nil
}

// ApplyDamageWithArmor applies armor mitigation before marking HP.
func ApplyDamageWithArmor(currentHP, currentArmor int, result DamageResult) DamageApplication {
	armorBefore := currentArmor
	reduced := result
	spent := 0
	if currentArmor > 0 {
		reduced, spent = ReduceDamageWithArmor(result, currentArmor)
		currentArmor -= spent
	}
	before, after := ApplyDamageMarks(currentHP, reduced.Marks)
	return DamageApplication{
		Result:      reduced,
		HPBefore:    before,
		HPAfter:     after,
		ArmorBefore: armorBefore,
		ArmorAfter:  currentArmor,
		ArmorSpent:  spent,
	}
}

// ReduceDamageWithArmor reduces damage severity by one step when armor is spent.
func ReduceDamageWithArmor(result DamageResult, availableSlots int) (DamageResult, int) {
	if availableSlots <= 0 {
		return result, 0
	}
	if result.Marks <= 0 {
		return result, 0
	}
	reduced := result
	if reduced.Severity > DamageNone {
		reduced.Severity--
	}
	reduced.Marks = reduced.Marks - 1
	if reduced.Marks < 0 {
		reduced.Marks = 0
	}
	return reduced, 1
}

// ApplyResistance adjusts damage based on resistance/immunity rules.
// Mixed damage only benefits from resistance if the target resists both types.
func ApplyResistance(amount int, types DamageTypes, resist ResistanceProfile) int {
	if amount <= 0 {
		return 0
	}
	if types.Physical && resist.ImmunePhysical {
		if !types.Magic || resist.ImmuneMagic {
			return 0
		}
	}
	if types.Magic && resist.ImmuneMagic {
		if !types.Physical || resist.ImmunePhysical {
			return 0
		}
	}

	if types.Physical && types.Magic {
		if resist.ResistPhysical && resist.ResistMagic {
			return amount / 2
		}
		return amount
	}
	if types.Physical {
		if resist.ResistPhysical {
			return amount / 2
		}
		return amount
	}
	if types.Magic {
		if resist.ResistMagic {
			return amount / 2
		}
		return amount
	}
	return amount
}
