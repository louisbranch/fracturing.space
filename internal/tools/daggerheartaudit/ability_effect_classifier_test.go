package main

import (
	"testing"
)

func TestClassifyAbilityEffectsEmptyText(t *testing.T) {
	c := classifyAbilityEffects("")
	if len(c.Effects) != 1 || c.Effects[0] != effectNarrative {
		t.Fatalf("Effects = %v, want [narrative_only]", c.Effects)
	}
	if c.Expressibility != notApplicable {
		t.Fatalf("Expressibility = %q, want not_applicable", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsDamageOnly(t *testing.T) {
	c := classifyAbilityEffects("Deal 2d6 magic damage to a nearby target.")
	assertContainsEffect(t, c.Effects, effectDamage)
	if c.Expressibility != expressible {
		t.Fatalf("Expressibility = %q, want expressible", c.Expressibility)
	}
	if len(c.Scenarios) == 0 {
		t.Fatal("expected scenario evidence for damage effect")
	}
}

func TestClassifyAbilityEffectsCondition(t *testing.T) {
	c := classifyAbilityEffects("The target becomes vulnerable until end of scene.")
	assertContainsEffect(t, c.Effects, effectCondition)
	assertContainsEffect(t, c.Effects, effectTempBuff)
}

func TestClassifyAbilityEffectsHealing(t *testing.T) {
	c := classifyAbilityEffects("Heal an ally within close range, restoring 1d6 hit points.")
	assertContainsEffect(t, c.Effects, effectHealing)
	if c.Expressibility != expressible {
		t.Fatalf("Expressibility = %q, want expressible", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsResourceGrant(t *testing.T) {
	c := classifyAbilityEffects("When you succeed on this roll, gain a Hope.")
	assertContainsEffect(t, c.Effects, effectResourceGrant)
	if c.Expressibility != expressible {
		t.Fatalf("Expressibility = %q, want expressible", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsStatOverride(t *testing.T) {
	c := classifyAbilityEffects("Your armor score increases by 2.")
	assertContainsEffect(t, c.Effects, effectStatOverride)
	if c.Expressibility != expressible {
		t.Fatalf("Expressibility = %q, want expressible", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsRollModification(t *testing.T) {
	c := classifyAbilityEffects("Roll with advantage on your next action roll.")
	assertContainsEffect(t, c.Effects, effectRollMod)
	if c.Expressibility == missingPrimitive {
		t.Fatalf("roll modification should not be missing_primitive")
	}
}

func TestClassifyAbilityEffectsMovement(t *testing.T) {
	c := classifyAbilityEffects("Teleport to a location within far range.")
	assertContainsEffect(t, c.Effects, effectMovement)
	if c.Expressibility != notApplicable {
		t.Fatalf("Expressibility = %q, want not_applicable (movement is non-goal)", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsNarrativeOnly(t *testing.T) {
	c := classifyAbilityEffects("You can speak to animals and understand their replies.")
	if len(c.Effects) != 1 || c.Effects[0] != effectNarrative {
		t.Fatalf("Effects = %v, want [narrative_only]", c.Effects)
	}
	if c.Expressibility != notApplicable {
		t.Fatalf("Expressibility = %q, want not_applicable", c.Expressibility)
	}
}

func TestClassifyAbilityEffectsMultiple(t *testing.T) {
	c := classifyAbilityEffects("Deal 1d8 damage and the target becomes restrained until rest.")
	assertContainsEffect(t, c.Effects, effectDamage)
	assertContainsEffect(t, c.Effects, effectCondition)
	assertContainsEffect(t, c.Effects, effectTempBuff)
}

func TestClassifyAbilityEffectsWallWalk(t *testing.T) {
	// Real domain card text: movement + resource spend + temp buff.
	// Movement is non-goal (notApplicable), resource_grant and temp_buff are
	// both expressible, so worst-case is expressible.
	c := classifyAbilityEffects("Spend a Hope to let a touched creature walk on walls and ceilings until the scene ends or you cast Wallwalk again.")
	assertContainsEffect(t, c.Effects, effectResourceGrant)
	assertContainsEffect(t, c.Effects, effectMovement)
	assertContainsEffect(t, c.Effects, effectTempBuff)
	if c.Expressibility != expressible {
		t.Fatalf("Expressibility = %q, want expressible", c.Expressibility)
	}
}

func TestDeriveExpressibilityWorstCase(t *testing.T) {
	// expressible + expressible → expressible (stat_override is now expressible via ApplyStatModifiers)
	result := deriveExpressibility([]abilityEffectCategory{effectDamage, effectStatOverride})
	if result != expressible {
		t.Fatalf("deriveExpressibility = %q, want expressible", result)
	}
}

func TestDeriveExpressibilityHealingNowExpressible(t *testing.T) {
	// damage + healing → both expressible → expressible
	result := deriveExpressibility([]abilityEffectCategory{effectDamage, effectHealing})
	if result != expressible {
		t.Fatalf("deriveExpressibility = %q, want expressible", result)
	}
}

func TestBuildAbilityAssessmentHealingNowCovered(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "ability-healing-touch", Kind: "ability", Title: "Healing Touch"}
	matches := map[string]abilityDomainCardMatch{
		"ability-healing-touch": {
			DomainCardID:   "domain_card.healing-touch",
			DomainCardName: "Healing Touch",
			DomainID:       "domain.grace",
			FeatureText:    "Heal a nearby ally, restoring 1d8 hit points.",
		},
	}
	assessment := buildAbilityAssessment(row, matches)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
	if assessment.GapClass != "" {
		t.Fatalf("GapClass = %q, want empty", assessment.GapClass)
	}
}

func TestBuildAbilityAssessmentMovementNonGoalClassified(t *testing.T) {
	// Movement is a non-goal (notApplicable), so movement-only abilities
	// are covered rather than gapped.
	row := auditMatrixRow{ReferenceID: "ability-dimension-door", Kind: "ability", Title: "Dimension Door"}
	matches := map[string]abilityDomainCardMatch{
		"ability-dimension-door": {
			DomainCardID:   "domain_card.dimension-door",
			DomainCardName: "Dimension Door",
			DomainID:       "domain.arcana",
			FeatureText:    "Teleport to a location within far range.",
		},
	}
	assessment := buildAbilityAssessment(row, matches)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
	if assessment.GapClass != "" {
		t.Fatalf("GapClass = %q, want empty", assessment.GapClass)
	}
}

func TestBuildAbilityAssessmentCoveredWhenExpressible(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "ability-fire-bolt", Kind: "ability", Title: "Fire Bolt"}
	matches := map[string]abilityDomainCardMatch{
		"ability-fire-bolt": {
			DomainCardID:   "domain_card.fire-bolt",
			DomainCardName: "Fire Bolt",
			DomainID:       "domain.arcana",
			FeatureText:    "Deal 2d6 magic damage to a target within far range.",
		},
	}
	assessment := buildAbilityAssessment(row, matches)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
	if assessment.GapClass != "" {
		t.Fatalf("GapClass = %q, want empty", assessment.GapClass)
	}
}

func TestBuildAbilityAssessmentCoveredWhenNarrative(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "ability-speak-to-animals", Kind: "ability", Title: "Speak to Animals"}
	matches := map[string]abilityDomainCardMatch{
		"ability-speak-to-animals": {
			DomainCardID:   "domain_card.speak-to-animals",
			DomainCardName: "Speak to Animals",
			DomainID:       "domain.sage",
			FeatureText:    "You can communicate with animals and understand their replies.",
		},
	}
	assessment := buildAbilityAssessment(row, matches)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
}

func assertContainsEffect(t *testing.T, effects []abilityEffectCategory, want abilityEffectCategory) {
	t.Helper()
	for _, e := range effects {
		if e == want {
			return
		}
	}
	t.Fatalf("effects %v does not contain %q", effects, want)
}
