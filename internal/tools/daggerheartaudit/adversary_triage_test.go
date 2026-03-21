package main

import "testing"

func TestParseAdversaryFeatures(t *testing.T) {
	content := `---
id: "adversary-test"
---

# Test Adversary

## Structured Data

- Difficulty: 14

## Feature

### Momentum - Reaction

On a successful attack, gain 1 Fear.

### Group Attack - Action

Roll damage for each contributor.

### Shield Wall - Passive

A creature who tries to move within range must succeed on a roll.

### Corrosive Blood - Reaction

When hit, all nearby creatures must mark an armor slot.
`
	features := parseAdversaryFeatures(content)
	if len(features) != 4 {
		t.Fatalf("len(features) = %d, want 4", len(features))
	}
	tests := []struct {
		name string
		kind string
	}{
		{"Momentum", "reaction"},
		{"Group Attack", "action"},
		{"Shield Wall", "passive"},
		{"Corrosive Blood", "reaction"},
	}
	for i, tt := range tests {
		if features[i].Name != tt.name {
			t.Errorf("features[%d].Name = %q, want %q", i, features[i].Name, tt.name)
		}
		if features[i].Kind != tt.kind {
			t.Errorf("features[%d].Kind = %q, want %q", i, features[i].Kind, tt.kind)
		}
	}
}

func TestParseAdversaryFeaturesRelentlessParenStripped(t *testing.T) {
	content := `## Feature

### Relentless (3) - Passive

Can be spotlighted up to three times.

### Flying - Passive

+2 Difficulty while airborne.
`
	features := parseAdversaryFeatures(content)
	if len(features) != 2 {
		t.Fatalf("len(features) = %d, want 2", len(features))
	}
	if features[0].Name != "Relentless" {
		t.Errorf("features[0].Name = %q, want Relentless", features[0].Name)
	}
	if features[1].Name != "Flying" {
		t.Errorf("features[1].Name = %q, want Flying", features[1].Name)
	}
}

func TestClassifyFeatureRuntimeSupported(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		wantRule string
	}{
		{"Momentum", "reaction", "momentum_gain_fear_on_successful_attack"},
		{"Terrifying", "passive", "terrifying_hope_loss_on_successful_attack"},
		{"Group Attack", "action", "group_attack"},
		{"Cloaked", "passive", "hidden_until_next_attack"},
		{"Flying", "passive", "difficulty_bonus_while_active"},
		{"Pack Tactics", "passive", "conditional_damage_replacement_with_contributor"},
		{"Box In", "passive", "focus_target_disadvantage"},
	}
	for _, tt := range tests {
		fc := classifyFeature(parsedFeature{Name: tt.name, Kind: tt.kind})
		if fc.Status != "runtime_supported" {
			t.Errorf("%s: status = %q, want runtime_supported", tt.name, fc.Status)
		}
		if fc.RuleKind != tt.wantRule {
			t.Errorf("%s: rule_kind = %q, want %q", tt.name, fc.RuleKind, tt.wantRule)
		}
	}
}

func TestClassifyFeatureArmorShredByDescription(t *testing.T) {
	fc := classifyFeature(parsedFeature{
		Name:        "Corrosive Blood",
		Kind:        "reaction",
		Description: "When hit, all nearby creatures must mark an armor slot.",
	})
	if fc.Status != "runtime_supported" {
		t.Errorf("status = %q, want runtime_supported", fc.Status)
	}
	if fc.RuleKind != "armor_shred_on_successful_attack" {
		t.Errorf("rule_kind = %q, want armor_shred_on_successful_attack", fc.RuleKind)
	}
}

func TestClassifyFeatureRecurringRule(t *testing.T) {
	for _, name := range []string{"Relentless", "Minion", "Horde"} {
		fc := classifyFeature(parsedFeature{Name: name, Kind: "passive"})
		if fc.Status != "recurring_rule" {
			t.Errorf("%s: status = %q, want recurring_rule", name, fc.Status)
		}
	}
}

func TestClassifyFeatureNarrative(t *testing.T) {
	tests := []struct {
		name string
		kind string
	}{
		{"Shield Wall", "passive"},
		{"Death Quake", "reaction"},
		{"Mind Dance", "action"},
	}
	for _, tt := range tests {
		fc := classifyFeature(parsedFeature{Name: tt.name, Kind: tt.kind})
		if fc.Status != "narrative" {
			t.Errorf("%s (%s): status = %q, want narrative", tt.name, tt.kind, fc.Status)
		}
	}
}

func TestAdversaryEntryClassAllCovered(t *testing.T) {
	cls := adversaryEntryClass{
		Features: []adversaryFeatureClass{
			{Status: "runtime_supported"},
			{Status: "recurring_rule"},
			{Status: "narrative"},
		},
		SupportedCount: 1,
		RecurringCount: 1,
		NarrativeCount: 1,
	}
	if !cls.allCovered() {
		t.Fatal("expected allCovered to be true")
	}
}

func TestBuildAdversaryAssessmentCoveredAllSupported(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "adversary-test", Kind: "adversary"}
	classes := map[string]adversaryEntryClass{
		"adversary-test": {
			Features: []adversaryFeatureClass{
				{Status: "runtime_supported"},
				{Status: "runtime_supported"},
			},
			SupportedCount: 2,
		},
	}
	assessment := buildAdversaryAssessment(row, classes)
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

func TestBuildAdversaryAssessmentCoveredMixed(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "adversary-test", Kind: "adversary"}
	classes := map[string]adversaryEntryClass{
		"adversary-test": {
			Features: []adversaryFeatureClass{
				{Status: "runtime_supported"},
				{Status: "recurring_rule"},
				{Status: "narrative"},
				{Status: "narrative"},
			},
			SupportedCount: 1,
			RecurringCount: 1,
			NarrativeCount: 2,
		},
	}
	assessment := buildAdversaryAssessment(row, classes)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
}

func TestBuildAdversaryAssessmentNoFeatures(t *testing.T) {
	row := auditMatrixRow{ReferenceID: "adversary-empty", Kind: "adversary"}
	assessment := buildAdversaryAssessment(row, nil)
	if assessment.FinalStatus != "covered" {
		t.Fatalf("FinalStatus = %q, want covered", assessment.FinalStatus)
	}
}
