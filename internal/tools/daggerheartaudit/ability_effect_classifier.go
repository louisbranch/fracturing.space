package main

import (
	"regexp"
	"strings"
)

// abilityEffectCategory names a mechanical effect family that a domain card's
// feature text may reference.
type abilityEffectCategory string

const (
	effectDamage        abilityEffectCategory = "damage"
	effectCondition     abilityEffectCategory = "condition"
	effectHealing       abilityEffectCategory = "healing"
	effectResourceGrant abilityEffectCategory = "resource_grant"
	effectStatOverride  abilityEffectCategory = "stat_override"
	effectRollMod       abilityEffectCategory = "roll_modification"
	effectTempBuff      abilityEffectCategory = "temporary_buff"
	effectMovement      abilityEffectCategory = "movement"
	effectNarrative     abilityEffectCategory = "narrative_only"
)

// abilityExpressibility records whether an effect category can be expressed
// through existing mutation primitives.
type abilityExpressibility string

const (
	expressible      abilityExpressibility = "expressible"
	missingPrimitive abilityExpressibility = "missing_primitive"
	notApplicable    abilityExpressibility = "not_applicable"
)

// abilityEffectClassification is the result of classifying a single domain
// card's feature text.
type abilityEffectClassification struct {
	Effects        []abilityEffectCategory
	Expressibility abilityExpressibility
	// Scenarios lists existing test scenario files that cover the relevant
	// mutation primitives exercised by this card's effects.
	Scenarios []string
}

// Compiled patterns for keyword-based classification.  Each pattern is
// case-insensitive and matches inside the feature_text string.
var (
	reDamage    = regexp.MustCompile(`(?i)\b(damage|d[0-9]+\s*(magic|physical|damage))\b`)
	reCondition = regexp.MustCompile(`(?i)\b(vulnerable|restrained|stunned|hidden|cloaked|condition|on fire)\b`)
	reHealing   = regexp.MustCompile(`(?i)\b(heal|restore.*h(it\s*)?p|clear.*hit\s*point|regain.*hit\s*point|recover.*h(it\s*)?p)\b`)
	reResource  = regexp.MustCompile(`(?i)\b(gain\s+(a\s+)?hope|grant\s+(a\s+)?hope|clear\s+(a\s+)?stress|reduce\s+(a\s+)?stress|spend\s+(a\s+)?hope)\b`)
	reStat      = regexp.MustCompile(`(?i)\b(armor\s+score|threshold|evasion|proficiency)\b`)
	reRollMod   = regexp.MustCompile(`(?i)\b(advantage|disadvantage|\+[0-9]+\s+to\s+.*roll|bonus\s+to\s+.*roll|roll\s+with\s+advantage)\b`)
	reTempBuff  = regexp.MustCompile(`(?i)\b(until\s+(your\s+next\s+)?rest|until\s+(the\s+)?scene|temporary|until\s+(the\s+)?end\s+of)\b`)
	reMovement  = regexp.MustCompile(`(?i)\b(teleport|push\s+(them|target|a\s+creature)|pull\s+(them|target|a\s+creature)|move\s+within|move\s+to\s+|walk\s+on\s+walls)\b`)
)

// classifyAbilityEffects parses a domain card's feature_text and returns the
// detected effect categories, overall expressibility, and relevant scenario
// paths.
func classifyAbilityEffects(featureText string) abilityEffectClassification {
	text := strings.TrimSpace(featureText)
	if text == "" {
		return abilityEffectClassification{
			Effects:        []abilityEffectCategory{effectNarrative},
			Expressibility: notApplicable,
		}
	}

	var effects []abilityEffectCategory
	var scenarios []string

	if reDamage.MatchString(text) {
		effects = append(effects, effectDamage)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/damage_thresholds_example.lua",
			"internal/test/game/scenarios/systems/daggerheart/combined_damage_sources.lua",
			"internal/test/game/scenarios/systems/daggerheart/critical_damage.lua",
		)
	}
	if reCondition.MatchString(text) {
		effects = append(effects, effectCondition)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/condition_lifecycle.lua",
			"internal/test/game/scenarios/systems/daggerheart/condition_stacking_guard.lua",
		)
	}
	if reHealing.MatchString(text) {
		effects = append(effects, effectHealing)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/rest_and_downtime.lua",
			"internal/test/game/scenarios/systems/daggerheart/death_move.lua",
		)
	}
	if reResource.MatchString(text) {
		effects = append(effects, effectResourceGrant)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/action_roll_failure_with_hope.lua",
			"internal/test/game/scenarios/systems/daggerheart/action_roll_outcomes.lua",
			"internal/test/game/scenarios/systems/daggerheart/action_roll_critical_success.lua",
			"internal/test/game/scenarios/systems/daggerheart/companion_experience_stress_clear.lua",
			"internal/test/game/scenarios/systems/daggerheart/spellcast_hope_cost.lua",
		)
	}
	if reStat.MatchString(text) {
		effects = append(effects, effectStatOverride)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/armor_swap_effective_stats.lua",
			"internal/test/game/scenarios/systems/daggerheart/stat_modifier_lifecycle.lua",
		)
	}
	if reRollMod.MatchString(text) {
		effects = append(effects, effectRollMod)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/advantage_cancellation.lua",
			"internal/test/game/scenarios/systems/daggerheart/damage_roll_modifier.lua",
		)
	}
	if reTempBuff.MatchString(text) {
		effects = append(effects, effectTempBuff)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/armor_feature_rules.lua",
			"internal/test/game/scenarios/systems/daggerheart/condition_lifecycle.lua",
			"internal/test/game/scenarios/systems/daggerheart/rest_and_downtime.lua",
		)
	}
	if reMovement.MatchString(text) {
		effects = append(effects, effectMovement)
	}

	if len(effects) == 0 {
		return abilityEffectClassification{
			Effects:        []abilityEffectCategory{effectNarrative},
			Expressibility: notApplicable,
		}
	}

	expr := deriveExpressibility(effects)
	return abilityEffectClassification{
		Effects:        effects,
		Expressibility: expr,
		Scenarios:      scenarios,
	}
}

// deriveExpressibility returns the worst-case expressibility across all
// detected effect categories.
func deriveExpressibility(effects []abilityEffectCategory) abilityExpressibility {
	worst := notApplicable
	for _, e := range effects {
		cat := categoryExpressibility(e)
		if precedence(cat) > precedence(worst) {
			worst = cat
		}
	}
	return worst
}

// categoryExpressibility maps each effect category to whether the existing
// mutation command surface can express it.
//
// Movement is classified as notApplicable (non-goal): no positioning model
// exists and one is out of scope for the foreseeable future.
//
// Temporary buffs are expressible via the condition system's ClearTrigger
// mechanism (short_rest, long_rest, scene_end).  Scene-end wiring is
// incremental follow-up work, not a missing primitive.
//
// Stat overrides (armor score, evasion, proficiency, thresholds) are
// expressible via the ApplyStatModifiers RPC which lets the GM add/remove
// structured stat modifiers to characters at runtime.
func categoryExpressibility(cat abilityEffectCategory) abilityExpressibility {
	switch cat {
	case effectDamage, effectCondition, effectRollMod, effectHealing, effectResourceGrant, effectTempBuff, effectStatOverride:
		return expressible
	case effectMovement:
		return notApplicable
	case effectNarrative:
		return notApplicable
	default:
		return notApplicable
	}
}

func precedence(e abilityExpressibility) int {
	switch e {
	case missingPrimitive:
		return 2
	case expressible:
		return 1
	case notApplicable:
		return 0
	default:
		return 0
	}
}

func appendUnique(slice []string, values ...string) []string {
	seen := make(map[string]struct{}, len(slice))
	for _, v := range slice {
		seen[v] = struct{}{}
	}
	for _, v := range values {
		if _, ok := seen[v]; !ok {
			slice = append(slice, v)
			seen[v] = struct{}{}
		}
	}
	return slice
}
