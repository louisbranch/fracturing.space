package catalogimporter

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func deriveArmorRules(item armorRecord) contentstore.DaggerheartArmorRules {
	rules := contentstore.DaggerheartArmorRules{
		AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
		MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
		SeverityReductionSteps: 1,
	}

	switch strings.TrimSpace(item.Feature) {
	case "":
		return rules
	case "Flexible: +1 to Evasion":
		rules.EvasionDelta = 1
		return rules
	case "Heavy: -1 to Evasion":
		rules.EvasionDelta = -1
		return rules
	case "Very Heavy: -2 to Evasion; -1 to Agility":
		rules.EvasionDelta = -2
		rules.AgilityDelta = -1
		return rules
	case "Gilded: +1 to Presence":
		rules.PresenceDelta = 1
		return rules
	case "Channeling: +1 to Spellcast Rolls":
		rules.SpellcastRollBonus = 1
		return rules
	case "Difficult: -1 to all character traits and Evasion":
		rules.AllTraitsDelta = -1
		rules.EvasionDelta = -1
		return rules
	case "Physical: You can't mark an Armor Slot to reduce magic damage.":
		rules.MitigationMode = contentstore.DaggerheartArmorMitigationModePhysicalOnly
		return rules
	case "Magic: You can't mark an Armor Slot to reduce physical damage.":
		rules.MitigationMode = contentstore.DaggerheartArmorMitigationModeMagicOnly
		return rules
	case "Warded: You reduce incoming magic damage by your Armor Score before applying it to your damage thresholds.":
		rules.WardedMagicReduction = true
		return rules
	case "Painful: Each time you mark an Armor Slot, you must mark a Stress.":
		rules.StressOnMark = true
		return rules
	case "Fortified: When you mark an Armor Slot, you reduce the severity of an attack by two thresholds instead of one.":
		rules.SeverityReductionSteps = 2
		return rules
	case "Reinforced: When you mark your last Armor Slot, increase your damage thresholds by +2 until you clear at least 1 Armor Slot.":
		rules.ThresholdBonusWhenArmorDepleted = 2
		return rules
	case "Hopeful: When you would spend a Hope, you can mark an Armor Slot instead.":
		rules.HopefulReplaceHopeWithArmor = true
		return rules
	case "Resilient: Before you mark your last Armor Slot, roll a d6. On a result of 6, reduce the severity by one threshold without marking an Armor Slot.":
		rules.ResilientDieSides = 6
		rules.ResilientSuccessOnOrAbove = 6
		return rules
	case "Shifting: When you are targeted for an attack, you can mark an Armor Slot to give the attack roll against you disadvantage.":
		rules.ShiftingAttackDisadvantage = 1
		return rules
	case "Timeslowing: Mark an Armor Slot to roll a d4 and add its result as a bonus to your Evasion against an incoming attack.":
		rules.TimeslowingEvasionBonusDieSides = 4
		return rules
	case "Sharp: On a successful attack against a target within Melee range, add a d4 to the damage roll.":
		rules.SharpDamageBonusDieSides = 4
		return rules
	case "Burning: When an adversary attacks you within Melee range, they mark a Stress.":
		rules.BurningAttackerStress = 1
		return rules
	case "Impenetrable: Once per short rest, when you would mark your last Hit Point, you can instead mark a Stress.":
		rules.ImpenetrableStressCost = 1
		rules.ImpenetrableUsesPerShortRest = 1
		return rules
	case "Quiet: You gain a +2 bonus to rolls you make to move silently.":
		rules.SilentMovementBonus = 2
		return rules
	default:
		rules.AutomationStatus = contentstore.DaggerheartArmorAutomationStatusUnsupported
		return rules
	}
}
