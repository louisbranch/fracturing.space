package main

import (
	"fmt"
	"regexp"
	"strings"
)

// itemEffectCategory names a mechanical effect family that an item or
// consumable's effect_text may reference.
type itemEffectCategory string

const (
	itemEffectHPRestore     itemEffectCategory = "hp_restore"
	itemEffectStressReduce  itemEffectCategory = "stress_reduce"
	itemEffectHopeGrant     itemEffectCategory = "hope_grant"
	itemEffectDamage        itemEffectCategory = "damage"
	itemEffectCondition     itemEffectCategory = "condition"
	itemEffectStatMod       itemEffectCategory = "stat_modifier"
	itemEffectRollMod       itemEffectCategory = "roll_modifier"
	itemEffectMovement      itemEffectCategory = "movement"
	itemEffectWeaponMod     itemEffectCategory = "weapon_modifier"
	itemEffectEquipmentStat itemEffectCategory = "equipment_stat"
	itemEffectDowntime      itemEffectCategory = "downtime"
	itemEffectNarrative     itemEffectCategory = "narrative"
)

// itemExpressibility records whether an item's effect can be expressed through
// existing mutation primitives.
type itemExpressibility string

const (
	itemExpressible   itemExpressibility = "expressible"
	itemNonGoal       itemExpressibility = "non_goal"
	itemNeedsNewModel itemExpressibility = "needs_new_model"
	itemNarrativeOnly itemExpressibility = "narrative_only"
)

// itemEffectClassification is the result of classifying a single item's
// effect_text.
type itemEffectClassification struct {
	Categories     []itemEffectCategory
	Expressibility itemExpressibility
	Scenarios      []string
}

// Compiled patterns for item/consumable effect classification.
var (
	reItemHP      = regexp.MustCompile(`(?i)\b(clear\s+\d+\S*\s+h(it\s*)?p|heal|restore.*h(it\s*)?p|clear.*h(it\s*)?p|regain.*h(it\s*)?p)\b`)
	reItemStress  = regexp.MustCompile(`(?i)\b(clear\s+(\d+\S*\s+|a\s+|all\s+)?stress|reduce.*stress)\b`)
	reItemHope    = regexp.MustCompile(`(?i)\b(gain\s+(a\s+|\d+\s+)?hope|grant\s+(a\s+)?hope|clear\s+(a\s+)?hope)\b`)
	reItemDamage  = regexp.MustCompile(`(?i)\b(\d+d\d+\s*(magic|physical)?\s*damage|deal\s+.*damage|take\s+.*damage)\b`)
	reItemCond    = regexp.MustCompile(`(?i)\b(vulnerable|restrained|hidden|cloaked|stunned|condition)\b`)
	reItemStatMod = regexp.MustCompile(`(?i)\b(\+\d+\s+(to\s+)?(strength|finesse|agility|instinct|presence|knowledge|proficiency)|instinct|strength|finesse|agility|presence|knowledge)\s+(rolls?|until)\b`)
	reItemRollMod = regexp.MustCompile(`(?i)\b(advantage|disadvantage|bonus\s+to\s+.*roll|\+\d+\s+to\s+.*roll)\b`)
	reItemMove    = regexp.MustCompile(`(?i)\b(teleport|reappear|fly|walk\s+on\s+water|climb|leap|glide|slow\s+fall)\b`)
	reItemWeapon  = regexp.MustCompile(`(?i)\b(weapon\s+feature|brutal|powerful|use\s+(strength|finesse|agility|instinct|presence|knowledge)\s+(for|on)\s+(attack|damage|this\s+weapon))\b`)
	reItemEquip   = regexp.MustCompile(`(?i)\b(\+\d+\s+(strength|finesse|agility|instinct|presence|knowledge|proficiency|experience)\b)`)
	reItemDown    = regexp.MustCompile(`(?i)\b(downtime\s+move|during\s+downtime|craft\s+a)\b`)
)

// classifyItemEffects parses an item's effect_text and returns the detected
// effect categories and overall expressibility.
func classifyItemEffects(effectText string) itemEffectClassification {
	text := strings.TrimSpace(effectText)
	if text == "" {
		return itemEffectClassification{
			Categories:     []itemEffectCategory{itemEffectNarrative},
			Expressibility: itemNarrativeOnly,
		}
	}

	var cats []itemEffectCategory
	var scenarios []string

	if reItemHP.MatchString(text) {
		cats = append(cats, itemEffectHPRestore)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/rest_and_downtime.lua",
		)
	}
	if reItemStress.MatchString(text) {
		cats = append(cats, itemEffectStressReduce)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/companion_experience_stress_clear.lua",
		)
	}
	if reItemHope.MatchString(text) {
		cats = append(cats, itemEffectHopeGrant)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/action_roll_failure_with_hope.lua",
		)
	}
	if reItemDamage.MatchString(text) {
		cats = append(cats, itemEffectDamage)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/damage_thresholds_example.lua",
		)
	}
	if reItemCond.MatchString(text) {
		cats = append(cats, itemEffectCondition)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/condition_lifecycle.lua",
		)
	}
	if reItemMove.MatchString(text) {
		cats = append(cats, itemEffectMovement)
	}
	if reItemStatMod.MatchString(text) {
		cats = append(cats, itemEffectStatMod)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/stat_modifier_action_roll_consumption.lua",
		)
	}
	if reItemRollMod.MatchString(text) {
		cats = append(cats, itemEffectRollMod)
		scenarios = appendUnique(scenarios,
			"internal/test/game/scenarios/systems/daggerheart/stat_modifier_trait_stacking.lua",
		)
	}
	if reItemWeapon.MatchString(text) {
		cats = append(cats, itemEffectWeaponMod)
	}
	if reItemEquip.MatchString(text) {
		cats = append(cats, itemEffectEquipmentStat)
	}
	if reItemDown.MatchString(text) {
		cats = append(cats, itemEffectDowntime)
	}

	if len(cats) == 0 {
		return itemEffectClassification{
			Categories:     []itemEffectCategory{itemEffectNarrative},
			Expressibility: itemNarrativeOnly,
		}
	}

	expr := deriveItemExpressibility(cats)
	return itemEffectClassification{
		Categories:     cats,
		Expressibility: expr,
		Scenarios:      scenarios,
	}
}

// deriveItemExpressibility returns the worst-case expressibility across all
// detected item effect categories.
func deriveItemExpressibility(cats []itemEffectCategory) itemExpressibility {
	worst := itemNarrativeOnly
	for _, c := range cats {
		e := itemCategoryExpressibility(c)
		if itemPrecedence(e) > itemPrecedence(worst) {
			worst = e
		}
	}
	return worst
}

// itemCategoryExpressibility maps each item effect category to whether the
// existing command surface can express it.
//
// HP restore, stress reduce, hope grant → CharacterStatePatch
// Damage → AdversaryDamageApply / MultiTargetDamageApply
// Condition → ConditionChange
// Stat modifiers → ApplyStatModifiers (base traits + derived stats)
// Roll modifiers → ApplyStatModifiers (trait modifiers affect rolls)
// Movement → non-goal (no positioning model)
// Weapon modifiers, equipment stats → need new model
// Downtime → partially expressible but economy is out of scope for now
func itemCategoryExpressibility(cat itemEffectCategory) itemExpressibility {
	switch cat {
	case itemEffectHPRestore, itemEffectStressReduce, itemEffectHopeGrant,
		itemEffectDamage, itemEffectCondition,
		itemEffectStatMod, itemEffectRollMod:
		return itemExpressible
	case itemEffectMovement:
		return itemNonGoal
	case itemEffectNarrative:
		return itemNarrativeOnly
	case itemEffectWeaponMod, itemEffectEquipmentStat, itemEffectDowntime:
		return itemNeedsNewModel
	default:
		return itemNarrativeOnly
	}
}

func itemPrecedence(e itemExpressibility) int {
	switch e {
	case itemNeedsNewModel:
		return 3
	case itemExpressible:
		return 2
	case itemNonGoal:
		return 1
	case itemNarrativeOnly:
		return 0
	default:
		return 0
	}
}

// buildItemAssessment classifies an item or consumable reference row by
// looking up its effect text and running the item effect classifier.
func buildItemAssessment(row auditMatrixRow, itemMatches map[string]itemEffectMatch) curatedAssessment {
	evidenceCode := []string{
		"api/proto/systems/daggerheart/v1/content.proto",
		"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
		"internal/tools/importer/content/daggerheart/v1/",
	}
	evidenceTests := []string{
		"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
		"internal/services/game/storage/sqlite/daggerheartcontent/store_content_test.go",
	}
	evidenceDocs := []string{
		"docs/product/daggerheart-PRD.md",
	}

	match, ok := itemMatches[row.ReferenceID]
	if !ok {
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "partial",
			FinalStatus:   "gap",
			GapClass:      "missing_model",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes: []string{
				fmt.Sprintf("%s row not found in import catalog.", row.Kind),
			},
			FollowUpEpic: "item-use-modeling",
		}
	}

	classification := classifyItemEffects(match.EffectText)

	baseNotes := []string{
		fmt.Sprintf("%s is cataloged through content and inventory surfaces.", match.Name),
	}
	effectNote := fmt.Sprintf("Detected effects: %s (expressibility: %s).",
		joinItemCategories(classification.Categories), string(classification.Expressibility))
	baseNotes = append(baseNotes, effectNote)

	for _, s := range classification.Scenarios {
		evidenceTests = appendUnique(evidenceTests, s)
	}

	switch classification.Expressibility {
	case itemNeedsNewModel:
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "partial",
			FinalStatus:   "gap",
			GapClass:      "missing_model",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes:         baseNotes,
			FollowUpEpic:  "item-use-modeling",
		}
	default:
		// expressible, non_goal, and narrative_only all resolve to covered.
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes:         baseNotes,
		}
	}
}

func joinItemCategories(cats []itemEffectCategory) string {
	parts := make([]string, len(cats))
	for i, c := range cats {
		parts[i] = string(c)
	}
	return strings.Join(parts, ", ")
}
