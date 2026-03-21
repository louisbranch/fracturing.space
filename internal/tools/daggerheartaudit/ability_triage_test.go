package main

import "testing"

func TestBuildAbilityAssessmentCoveredWhenDamageOnly(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "ability-fire-blast", Kind: "ability", Title: "Fire Blast"}
	matches := map[string]abilityDomainCardMatch{
		"ability-fire-blast": {
			DomainCardID:   "domain_card.fire-blast",
			DomainCardName: "Fire Blast",
			DomainID:       "domain.arcana",
			FeatureText:    "Deal 2d8 magic damage to a target.",
		},
	}
	assessment := buildAbilityAssessment(row, matches)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
	if assessment.GapClass != "" {
		t.Fatalf("GapClass = %q, want empty", assessment.GapClass)
	}
	if assessment.FollowUpEpic != "" {
		t.Fatalf("FollowUpEpic = %q, want empty", assessment.FollowUpEpic)
	}
}

func TestBuildAbilityAssessmentMovementNonGoalIsCovered(t *testing.T) {
	// Movement is a non-goal (notApplicable), so movement-only cards resolve
	// to covered rather than gap.
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

func TestBuildAbilityAssessmentGapWhenUnmatched(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "ability-unknown-spell", Kind: "ability", Title: "Unknown Spell"}
	assessment := buildAbilityAssessment(row, nil)
	if assessment.FinalStatus != "gap" {
		t.Fatalf("FinalStatus = %q, want gap", assessment.FinalStatus)
	}
	if assessment.GapClass != "ambiguous_mapping" {
		t.Fatalf("GapClass = %q, want ambiguous_mapping", assessment.GapClass)
	}
	if assessment.FollowUpEpic != "ability-mapping-and-semantic-audit" {
		t.Fatalf("FollowUpEpic = %q, want ability-mapping-and-semantic-audit", assessment.FollowUpEpic)
	}
}
