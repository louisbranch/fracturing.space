package rules

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// EffectiveArmorRules returns runtime-safe armor rules with the default armor
// behavior filled in for entries that have no special automation.
func EffectiveArmorRules(armor *contentstore.DaggerheartArmor) contentstore.DaggerheartArmorRules {
	rules := contentstore.DaggerheartArmorRules{
		AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
		MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
		SeverityReductionSteps: 1,
	}
	if armor == nil {
		return rules
	}
	if strings.TrimSpace(string(armor.Rules.AutomationStatus)) != "" {
		rules.AutomationStatus = armor.Rules.AutomationStatus
	}
	if strings.TrimSpace(string(armor.Rules.MitigationMode)) != "" {
		rules.MitigationMode = armor.Rules.MitigationMode
	}
	if armor.Rules.SeverityReductionSteps > 0 {
		rules.SeverityReductionSteps = armor.Rules.SeverityReductionSteps
	}
	rules.EvasionDelta = armor.Rules.EvasionDelta
	rules.AgilityDelta = armor.Rules.AgilityDelta
	rules.PresenceDelta = armor.Rules.PresenceDelta
	rules.SpellcastRollBonus = armor.Rules.SpellcastRollBonus
	rules.AllTraitsDelta = armor.Rules.AllTraitsDelta
	rules.StressOnMark = armor.Rules.StressOnMark
	rules.ThresholdBonusWhenArmorDepleted = armor.Rules.ThresholdBonusWhenArmorDepleted
	rules.WardedMagicReduction = armor.Rules.WardedMagicReduction
	rules.HopefulReplaceHopeWithArmor = armor.Rules.HopefulReplaceHopeWithArmor
	rules.ResilientDieSides = armor.Rules.ResilientDieSides
	rules.ResilientSuccessOnOrAbove = armor.Rules.ResilientSuccessOnOrAbove
	rules.ShiftingAttackDisadvantage = armor.Rules.ShiftingAttackDisadvantage
	rules.TimeslowingEvasionBonusDieSides = armor.Rules.TimeslowingEvasionBonusDieSides
	rules.SharpDamageBonusDieSides = armor.Rules.SharpDamageBonusDieSides
	rules.BurningAttackerStress = armor.Rules.BurningAttackerStress
	rules.ImpenetrableStressCost = armor.Rules.ImpenetrableStressCost
	rules.ImpenetrableUsesPerShortRest = armor.Rules.ImpenetrableUsesPerShortRest
	rules.SilentMovementBonus = armor.Rules.SilentMovementBonus
	return rules
}

// ArmorPassiveBase captures armorless profile values for armor swap computation.
type ArmorPassiveBase struct {
	Evasion int
	Traits  daggerheartprofile.Traits
}

// RemoveArmorPassiveEffects recovers the armorless profile values from the
// currently stored effective profile and its equipped armor.
func RemoveArmorPassiveEffects(profile projectionstore.DaggerheartCharacterProfile, currentArmor *contentstore.DaggerheartArmor) ArmorPassiveBase {
	base := ArmorPassiveBase{
		Evasion: profile.Evasion,
		Traits: daggerheartprofile.Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
	}
	if currentArmor == nil {
		return base
	}
	rules := EffectiveArmorRules(currentArmor)
	base.Evasion -= rules.EvasionDelta
	base.Traits.Agility -= rules.AllTraitsDelta + rules.AgilityDelta
	base.Traits.Strength -= rules.AllTraitsDelta
	base.Traits.Finesse -= rules.AllTraitsDelta
	base.Traits.Instinct -= rules.AllTraitsDelta
	base.Traits.Presence -= rules.AllTraitsDelta + rules.PresenceDelta
	base.Traits.Knowledge -= rules.AllTraitsDelta
	return base
}

// ApplyArmorProfileEffects computes the effective stored profile values for one
// active armor entry from armorless base values.
func ApplyArmorProfileEffects(level int, base ArmorPassiveBase, armor *contentstore.DaggerheartArmor) projectionstore.DaggerheartCharacterProfile {
	result := projectionstore.DaggerheartCharacterProfile{
		Evasion:    base.Evasion,
		ArmorScore: 0,
		ArmorMax:   0,
		Agility:    base.Traits.Agility,
		Strength:   base.Traits.Strength,
		Finesse:    base.Traits.Finesse,
		Instinct:   base.Traits.Instinct,
		Presence:   base.Traits.Presence,
		Knowledge:  base.Traits.Knowledge,
	}
	if armor == nil {
		result.MajorThreshold, result.SevereThreshold = daggerheartprofile.UnarmoredThresholds(level)
		return result
	}
	rules := EffectiveArmorRules(armor)
	result.Evasion += rules.EvasionDelta
	result.Agility += rules.AllTraitsDelta + rules.AgilityDelta
	result.Strength += rules.AllTraitsDelta
	result.Finesse += rules.AllTraitsDelta
	result.Instinct += rules.AllTraitsDelta
	result.Presence += rules.AllTraitsDelta + rules.PresenceDelta
	result.Knowledge += rules.AllTraitsDelta
	result.SpellcastRollBonus = rules.SpellcastRollBonus
	result.EquippedArmorID = strings.TrimSpace(armor.ID)
	result.ArmorScore = armor.ArmorScore
	result.ArmorMax = armor.ArmorScore
	result.MajorThreshold, result.SevereThreshold = daggerheartprofile.DeriveThresholds(
		level,
		armor.ArmorScore,
		armor.BaseMajorThreshold,
		armor.BaseSevereThreshold,
	)
	return result
}

// RemapArmorCurrent preserves the number of marked base armor slots when a
// character changes their equipped armor. Temporary armor remains untouched.
func RemapArmorCurrent(state projectionstore.DaggerheartCharacterState, oldArmorMax, newArmorMax int) int {
	temporaryArmor := TemporaryArmorAmount(state)
	currentBaseArmor := state.Armor - temporaryArmor
	if currentBaseArmor < 0 {
		currentBaseArmor = 0
	}
	if currentBaseArmor > oldArmorMax {
		currentBaseArmor = oldArmorMax
	}
	marked := oldArmorMax - currentBaseArmor
	if marked < 0 {
		marked = 0
	}
	newBaseArmor := newArmorMax - marked
	if newBaseArmor < 0 {
		newBaseArmor = 0
	}
	if newBaseArmor > newArmorMax {
		newBaseArmor = newArmorMax
	}
	return newBaseArmor + temporaryArmor
}

// TemporaryArmorAmount returns the currently active temporary armor total.
func TemporaryArmorAmount(state projectionstore.DaggerheartCharacterState) int {
	total := 0
	for _, bucket := range state.TemporaryArmor {
		if bucket.Amount > 0 {
			total += bucket.Amount
		}
	}
	return total
}

// CurrentBaseArmor returns the current equipped-armor slots excluding temporary armor.
func CurrentBaseArmor(state projectionstore.DaggerheartCharacterState, armorMax int) int {
	baseArmor := state.Armor - TemporaryArmorAmount(state)
	if baseArmor < 0 {
		baseArmor = 0
	}
	if armorMax > 0 && baseArmor > armorMax {
		baseArmor = armorMax
	}
	return baseArmor
}

// SpendBaseArmorSlot marks one equipped-armor slot while preserving temporary armor.
func SpendBaseArmorSlot(state projectionstore.DaggerheartCharacterState, armorMax int) (before, after int, ok bool) {
	before = CurrentBaseArmor(state, armorMax)
	if before <= 0 {
		return before, before, false
	}
	after = before - 1
	return before, after, true
}

// ArmorTotalAfterBaseSpend returns the persisted armor total after consuming
// one equipped-armor slot while preserving temporary armor.
func ArmorTotalAfterBaseSpend(state projectionstore.DaggerheartCharacterState, armorMax int) (before, after int, ok bool) {
	baseBefore, baseAfter, ok := SpendBaseArmorSlot(state, armorMax)
	if !ok {
		return state.Armor, state.Armor, false
	}
	temporaryArmor := TemporaryArmorAmount(state)
	return baseBefore + temporaryArmor, baseAfter + temporaryArmor, true
}

// IsLastBaseArmorSlot reports whether only one equipped-armor slot remains.
func IsLastBaseArmorSlot(state projectionstore.DaggerheartCharacterState, armorMax int) bool {
	return CurrentBaseArmor(state, armorMax) == 1
}
