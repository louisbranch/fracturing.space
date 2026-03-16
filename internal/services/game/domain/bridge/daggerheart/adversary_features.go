package daggerheart

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

// AdversaryFeatureAutomationStatus captures whether a catalog adversary
// feature maps to a supported typed runtime family.
type AdversaryFeatureAutomationStatus string

const (
	AdversaryFeatureAutomationStatusUnspecified AdversaryFeatureAutomationStatus = ""
	AdversaryFeatureAutomationStatusSupported   AdversaryFeatureAutomationStatus = "supported"
	AdversaryFeatureAutomationStatusUnsupported AdversaryFeatureAutomationStatus = "unsupported"
)

// AdversaryFeatureRuleKind identifies one repeatable adversary feature family
// the runtime can execute or stage explicitly.
type AdversaryFeatureRuleKind string

const (
	AdversaryFeatureRuleKindUnspecified                                 AdversaryFeatureRuleKind = ""
	AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack          AdversaryFeatureRuleKind = "momentum_gain_fear_on_successful_attack"
	AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack        AdversaryFeatureRuleKind = "terrifying_hope_loss_on_successful_attack"
	AdversaryFeatureRuleKindGroupAttack                                 AdversaryFeatureRuleKind = "group_attack"
	AdversaryFeatureRuleKindHiddenUntilNextAttack                       AdversaryFeatureRuleKind = "hidden_until_next_attack"
	AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack         AdversaryFeatureRuleKind = "damage_replacement_on_advantaged_attack"
	AdversaryFeatureRuleKindDifficultyBonusWhileActive                  AdversaryFeatureRuleKind = "difficulty_bonus_while_active"
	AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor AdversaryFeatureRuleKind = "conditional_damage_replacement_with_contributor"
	AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack                AdversaryFeatureRuleKind = "armor_shred_on_successful_attack"
	AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit                 AdversaryFeatureRuleKind = "retaliatory_damage_on_close_hit"
	AdversaryFeatureRuleKindFocusTargetDisadvantage                     AdversaryFeatureRuleKind = "focus_target_disadvantage"
)

// AdversaryFeatureRule stores the typed recurring runtime behavior derived
// from an adversary feature description.
type AdversaryFeatureRule struct {
	Kind                AdversaryFeatureRuleKind
	FearGain            int
	HopeLoss            int
	DifficultyBonus     int
	DamageDice          []contentstore.DaggerheartDamageDie
	DamageBonus         int
	DamageType          string
	RequiresAdvantage   bool
	RequiresContributor bool
}

// AdversaryFeatureState stores mutable adversary feature runtime state.
type AdversaryFeatureState struct {
	FeatureID       string `json:"feature_id"`
	Status          string `json:"status,omitempty"`
	FocusedTargetID string `json:"focused_target_id,omitempty"`
}

// AdversaryPendingExperience stores one staged adversary experience modifier.
type AdversaryPendingExperience struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}

var (
	damageReplacementRegexp = regexp.MustCompile(`(\d+)d(\d+)\+(\d+)`)
	difficultyBonusRegexp   = regexp.MustCompile(`\+(\d+)\s+Difficulty`)
)

// ResolveAdversaryFeatureRuntime derives a supported runtime family from the
// imported adversary feature metadata when one matches a repeatable pattern.
func ResolveAdversaryFeatureRuntime(feature contentstore.DaggerheartAdversaryFeature) (AdversaryFeatureAutomationStatus, *AdversaryFeatureRule) {
	name := strings.ToLower(strings.TrimSpace(feature.Name))
	description := strings.ToLower(strings.TrimSpace(feature.Description))

	switch name {
	case "momentum":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind:     AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack,
			FearGain: 1,
		}
	case "terrifying":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind:     AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack,
			FearGain: 1,
			HopeLoss: 1,
		}
	case "group attack":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind: AdversaryFeatureRuleKindGroupAttack,
		}
	case "cloaked":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind: AdversaryFeatureRuleKindHiddenUntilNextAttack,
		}
	case "backstab":
		rule := &AdversaryFeatureRule{
			Kind:              AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack,
			RequiresAdvantage: true,
		}
		populateDamageReplacement(rule, description)
		return AdversaryFeatureAutomationStatusSupported, rule
	case "pack tactics":
		rule := &AdversaryFeatureRule{
			Kind:                AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor,
			RequiresContributor: true,
		}
		populateDamageReplacement(rule, description)
		return AdversaryFeatureAutomationStatusSupported, rule
	case "flying":
		rule := &AdversaryFeatureRule{
			Kind: AdversaryFeatureRuleKindDifficultyBonusWhileActive,
		}
		rule.DifficultyBonus = firstIntMatch(description, difficultyBonusRegexp)
		if rule.DifficultyBonus == 0 {
			rule.DifficultyBonus = 2
		}
		return AdversaryFeatureAutomationStatusSupported, rule
	case "warding sphere":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind:       AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit,
			DamageDice: []contentstore.DaggerheartDamageDie{{Count: 2, Sides: 6}},
			DamageType: "magic",
		}
	case "box in":
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind: AdversaryFeatureRuleKindFocusTargetDisadvantage,
		}
	}

	if strings.Contains(description, "mark an armor slot") {
		return AdversaryFeatureAutomationStatusSupported, &AdversaryFeatureRule{
			Kind: AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack,
		}
	}

	return AdversaryFeatureAutomationStatusUnsupported, nil
}

func populateDamageReplacement(rule *AdversaryFeatureRule, description string) {
	matches := damageReplacementRegexp.FindStringSubmatch(description)
	if len(matches) != 4 {
		return
	}
	count, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])
	bonus, _ := strconv.Atoi(matches[3])
	rule.DamageDice = []contentstore.DaggerheartDamageDie{{Count: count, Sides: sides}}
	rule.DamageBonus = bonus
}

func firstIntMatch(value string, pattern *regexp.Regexp) int {
	matches := pattern.FindStringSubmatch(value)
	if len(matches) != 2 {
		return 0
	}
	parsed, _ := strconv.Atoi(matches[1])
	return parsed
}
