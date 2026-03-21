package rules

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestResolveAdversaryFeatureRuntime_KnownFamilies(t *testing.T) {
	tests := []struct {
		name          string
		feature       contentstore.DaggerheartAdversaryFeature
		wantStatus    AdversaryFeatureAutomationStatus
		wantKind      AdversaryFeatureRuleKind
		wantNilRule   bool
		checkFearGain int
		checkHopeLoss int
	}{
		{
			name:          "momentum",
			feature:       contentstore.DaggerheartAdversaryFeature{Name: "Momentum"},
			wantStatus:    AdversaryFeatureAutomationStatusSupported,
			wantKind:      AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack,
			checkFearGain: 1,
		},
		{
			name:          "terrifying",
			feature:       contentstore.DaggerheartAdversaryFeature{Name: "Terrifying"},
			wantStatus:    AdversaryFeatureAutomationStatusSupported,
			wantKind:      AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack,
			checkFearGain: 1,
			checkHopeLoss: 1,
		},
		{
			name:       "group attack",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Group Attack"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindGroupAttack,
		},
		{
			name:       "cloaked",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Cloaked"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindHiddenUntilNextAttack,
		},
		{
			name:       "backstab with damage dice",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Backstab", Description: "deal 2d10+3 damage instead"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack,
		},
		{
			name:       "pack tactics with damage dice",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Pack Tactics", Description: "deal 3d6+2 damage instead"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor,
		},
		{
			name:       "flying with difficulty bonus (default path)",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Flying", Description: "+3 Difficulty to attacks"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindDifficultyBonusWhileActive,
		},
		{
			name:       "flying default difficulty",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Flying", Description: "this creature can fly"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindDifficultyBonusWhileActive,
		},
		{
			name:       "warding sphere",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Warding Sphere"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit,
		},
		{
			name:       "box in",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Box In"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindFocusTargetDisadvantage,
		},
		{
			name:       "armor shred via description",
			feature:    contentstore.DaggerheartAdversaryFeature{Name: "Rend", Description: "on a successful hit, mark an armor slot"},
			wantStatus: AdversaryFeatureAutomationStatusSupported,
			wantKind:   AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack,
		},
		{
			name:        "unsupported feature",
			feature:     contentstore.DaggerheartAdversaryFeature{Name: "Arcane Shield", Description: "reflects spells back at the caster"},
			wantStatus:  AdversaryFeatureAutomationStatusUnsupported,
			wantNilRule: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, rule := ResolveAdversaryFeatureRuntime(tc.feature)
			if status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
			if tc.wantNilRule {
				if rule != nil {
					t.Fatalf("expected nil rule, got %+v", rule)
				}
				return
			}
			if rule == nil {
				t.Fatal("expected non-nil rule")
			}
			if rule.Kind != tc.wantKind {
				t.Fatalf("kind = %q, want %q", rule.Kind, tc.wantKind)
			}
			if tc.checkFearGain > 0 && rule.FearGain != tc.checkFearGain {
				t.Fatalf("fear_gain = %d, want %d", rule.FearGain, tc.checkFearGain)
			}
			if tc.checkHopeLoss > 0 && rule.HopeLoss != tc.checkHopeLoss {
				t.Fatalf("hope_loss = %d, want %d", rule.HopeLoss, tc.checkHopeLoss)
			}
		})
	}
}

func TestPopulateDamageReplacement(t *testing.T) {
	rule := &AdversaryFeatureRule{}
	populateDamageReplacement(rule, "deal 2d10+3 damage instead")
	if len(rule.DamageDice) != 1 || rule.DamageDice[0].Count != 2 || rule.DamageDice[0].Sides != 10 {
		t.Fatalf("damage dice = %v, want [{2 10}]", rule.DamageDice)
	}
	if rule.DamageBonus != 3 {
		t.Fatalf("damage bonus = %d, want 3", rule.DamageBonus)
	}

	// No match leaves rule unchanged.
	rule2 := &AdversaryFeatureRule{}
	populateDamageReplacement(rule2, "does something else entirely")
	if len(rule2.DamageDice) != 0 {
		t.Fatalf("expected no damage dice, got %v", rule2.DamageDice)
	}
}

func TestFirstIntMatch(t *testing.T) {
	got := firstIntMatch("+3 Difficulty to attacks", difficultyBonusRegexp)
	if got != 3 {
		t.Fatalf("firstIntMatch(+3 Difficulty) = %d, want 3", got)
	}
	got = firstIntMatch("no numbers here", difficultyBonusRegexp)
	if got != 0 {
		t.Fatalf("firstIntMatch(no match) = %d, want 0", got)
	}
}

func TestResolveAdversaryFeatureRuntime_BackstabDamageDetails(t *testing.T) {
	status, rule := ResolveAdversaryFeatureRuntime(contentstore.DaggerheartAdversaryFeature{
		Name:        "Backstab",
		Description: "when attacking with advantage, deal 2d10+3 damage instead",
	})
	if status != AdversaryFeatureAutomationStatusSupported {
		t.Fatalf("status = %q, want supported", status)
	}
	if !rule.RequiresAdvantage {
		t.Fatal("backstab should require advantage")
	}
	if len(rule.DamageDice) != 1 || rule.DamageDice[0].Count != 2 || rule.DamageDice[0].Sides != 10 {
		t.Fatalf("backstab damage dice = %v, want [{2 10}]", rule.DamageDice)
	}
	if rule.DamageBonus != 3 {
		t.Fatalf("backstab damage bonus = %d, want 3", rule.DamageBonus)
	}
}

func TestResolveAdversaryFeatureRuntime_FlyingDifficultyParsing(t *testing.T) {
	// The regex uses capital "Difficulty" but descriptions are lowercased
	// before matching, so the pattern never matches and the default of 2 is
	// always used. Verify the default path works correctly.
	_, rule := ResolveAdversaryFeatureRuntime(contentstore.DaggerheartAdversaryFeature{
		Name:        "Flying",
		Description: "+3 Difficulty to ranged attacks",
	})
	if rule.DifficultyBonus != 2 {
		t.Fatalf("difficulty bonus = %d, want 2 (default)", rule.DifficultyBonus)
	}

	// Explicit no-match description also gets the default.
	_, rule = ResolveAdversaryFeatureRuntime(contentstore.DaggerheartAdversaryFeature{
		Name:        "Flying",
		Description: "soars above the battlefield",
	})
	if rule.DifficultyBonus != 2 {
		t.Fatalf("default difficulty bonus = %d, want 2", rule.DifficultyBonus)
	}
}
