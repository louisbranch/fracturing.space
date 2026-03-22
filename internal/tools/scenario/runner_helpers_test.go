package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ---------------------------------------------------------------------------
// Pure function tests — no Runner or gRPC fakes needed
// ---------------------------------------------------------------------------

func TestApplyTraitValue(t *testing.T) {
	traits := []string{"agility", "strength", "finesse", "instinct", "presence", "knowledge"}
	for _, trait := range traits {
		t.Run(trait, func(t *testing.T) {
			profile := &daggerheartv1.DaggerheartProfile{}
			args := map[string]any{trait: 3}
			applyTraitValue(profile, trait, args)

			var got int32
			switch trait {
			case "agility":
				got = profile.GetAgility().GetValue()
			case "strength":
				got = profile.GetStrength().GetValue()
			case "finesse":
				got = profile.GetFinesse().GetValue()
			case "instinct":
				got = profile.GetInstinct().GetValue()
			case "presence":
				got = profile.GetPresence().GetValue()
			case "knowledge":
				got = profile.GetKnowledge().GetValue()
			}
			if got != 3 {
				t.Fatalf("want 3, got %d", got)
			}
		})
	}
}

func TestApplyTraitValue_ZeroSkips(t *testing.T) {
	profile := &daggerheartv1.DaggerheartProfile{}
	applyTraitValue(profile, "agility", map[string]any{"agility": 0})
	if profile.Agility != nil {
		t.Fatal("expected nil agility for value=0")
	}
}

func TestParseDamageType(t *testing.T) {
	tests := []struct {
		input string
		want  daggerheartv1.DaggerheartDamageType
	}{
		{"physical", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL},
		{"PHYSICAL", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL},
		{"magic", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC},
		{"mixed", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED},
		{"unknown", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL},
		{"", daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseDamageType(tc.input)
			if got != tc.want {
				t.Fatalf("parseDamageType(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestDamageTypesForArgs(t *testing.T) {
	t.Run("default_physical", func(t *testing.T) {
		dt := damageTypesForArgs(map[string]any{})
		if !dt.Physical || dt.Magic {
			t.Fatalf("expected Physical=true,Magic=false, got %+v", dt)
		}
	})
	t.Run("magic", func(t *testing.T) {
		dt := damageTypesForArgs(map[string]any{"damage_type": "magic"})
		if dt.Physical || !dt.Magic {
			t.Fatalf("expected Physical=false,Magic=true, got %+v", dt)
		}
	})
	t.Run("mixed", func(t *testing.T) {
		dt := damageTypesForArgs(map[string]any{"damage_type": "mixed"})
		if !dt.Physical || !dt.Magic {
			t.Fatalf("expected Physical=true,Magic=true, got %+v", dt)
		}
	})
}

func TestBuildDamageSpec_Defaults(t *testing.T) {
	spec := buildDamageSpec(map[string]any{}, "actor-1", "sword")
	if spec.Source != "sword" {
		t.Fatalf("source = %q, want sword", spec.Source)
	}
	if len(spec.SourceCharacterIds) != 1 || spec.SourceCharacterIds[0] != "actor-1" {
		t.Fatalf("source_character_ids = %v, want [actor-1]", spec.SourceCharacterIds)
	}
	if spec.DamageType != daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL {
		t.Fatalf("damage_type = %v, want PHYSICAL", spec.DamageType)
	}
}

func TestBuildDamageSpec_Flags(t *testing.T) {
	args := map[string]any{
		"resist_physical": true,
		"immune_magic":    true,
		"direct":          true,
		"massive_damage":  true,
	}
	spec := buildDamageSpec(args, "", "")
	if !spec.ResistPhysical {
		t.Fatal("expected resist_physical")
	}
	if !spec.ImmuneMagic {
		t.Fatal("expected immune_magic")
	}
	if !spec.Direct {
		t.Fatal("expected direct")
	}
	if !spec.MassiveDamage {
		t.Fatal("expected massive_damage")
	}
}

func TestBuildDamageRequest(t *testing.T) {
	req := buildDamageRequest(map[string]any{"damage_type": "magic"}, "a-1", "spell", 10)
	if req.Amount != 10 {
		t.Fatalf("amount = %d, want 10", req.Amount)
	}
	if req.Source != "spell" {
		t.Fatalf("source = %q, want spell", req.Source)
	}
	if req.DamageType != daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC {
		t.Fatalf("damage_type = %v, want MAGIC", req.DamageType)
	}
}

func TestBuildDamageRequestWithSources(t *testing.T) {
	req := buildDamageRequestWithSources(map[string]any{}, "axe", 5, []string{"a", "b", "a", ""})
	if len(req.SourceCharacterIds) != 2 {
		t.Fatalf("expected 2 unique source IDs, got %v", req.SourceCharacterIds)
	}
}

func TestAdjustedDamageAmount(t *testing.T) {
	t.Run("no_resistance", func(t *testing.T) {
		got := adjustedDamageAmount(map[string]any{}, 10)
		if got != 10 {
			t.Fatalf("want 10, got %d", got)
		}
	})
	t.Run("resist_halves", func(t *testing.T) {
		got := adjustedDamageAmount(map[string]any{"resist_physical": true}, 10)
		if got != 5 {
			t.Fatalf("want 5 (halved), got %d", got)
		}
	})
	t.Run("immune_zeroes", func(t *testing.T) {
		got := adjustedDamageAmount(map[string]any{"immune_physical": true}, 10)
		if got != 0 {
			t.Fatalf("want 0 (immune), got %d", got)
		}
	})
}

func TestExpectDamageEffect(t *testing.T) {
	t.Run("nil_roll", func(t *testing.T) {
		if expectDamageEffect(map[string]any{}, nil) {
			t.Fatal("expected false for nil roll")
		}
	})
	t.Run("positive_damage", func(t *testing.T) {
		roll := &daggerheartv1.SessionDamageRollResponse{Total: 5}
		if !expectDamageEffect(map[string]any{}, roll) {
			t.Fatal("expected true for positive damage")
		}
	})
	t.Run("immune_no_effect", func(t *testing.T) {
		roll := &daggerheartv1.SessionDamageRollResponse{Total: 5}
		if expectDamageEffect(map[string]any{"immune_physical": true}, roll) {
			t.Fatal("expected false when immune")
		}
	})
}

func TestResolveOutcomeTargets(t *testing.T) {
	state := &scenarioState{actors: map[string]string{"Alice": "a-1", "Bob": "b-1"}}

	t.Run("empty", func(t *testing.T) {
		ids, err := resolveOutcomeTargets(state, map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ids != nil {
			t.Fatalf("expected nil, got %v", ids)
		}
	})
	t.Run("single_target", func(t *testing.T) {
		ids, err := resolveOutcomeTargets(state, map[string]any{"target": "Alice"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "a-1" {
			t.Fatalf("want [a-1], got %v", ids)
		}
	})
	t.Run("multiple_targets", func(t *testing.T) {
		ids, err := resolveOutcomeTargets(state, map[string]any{"targets": []any{"Alice", "Bob"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 {
			t.Fatalf("want 2 targets, got %v", ids)
		}
	})
	t.Run("unknown_error", func(t *testing.T) {
		_, err := resolveOutcomeTargets(state, map[string]any{"target": "Charlie"})
		if err == nil {
			t.Fatal("expected error for unknown actor")
		}
	})
}

func TestResolveAttackTargets(t *testing.T) {
	state := &scenarioState{
		actors:      map[string]string{"Alice": "a-1"},
		adversaries: map[string]string{"Dragon": "d-1"},
	}

	t.Run("actor_target", func(t *testing.T) {
		ids, err := resolveAttackTargets(state, map[string]any{"target": "Alice"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "a-1" {
			t.Fatalf("want [a-1], got %v", ids)
		}
	})
	t.Run("adversary_target", func(t *testing.T) {
		ids, err := resolveAttackTargets(state, map[string]any{"target": "Dragon"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "d-1" {
			t.Fatalf("want [d-1], got %v", ids)
		}
	})
	t.Run("unknown_error", func(t *testing.T) {
		_, err := resolveAttackTargets(state, map[string]any{"target": "Ghost"})
		if err == nil {
			t.Fatal("expected error for unknown target")
		}
	})
}

func TestResolveCharacterList(t *testing.T) {
	state := &scenarioState{actors: map[string]string{"Alice": "a-1", "Bob": "b-1"}}

	t.Run("empty", func(t *testing.T) {
		ids, err := resolveCharacterList(state, map[string]any{}, "characters")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ids != nil {
			t.Fatalf("expected nil, got %v", ids)
		}
	})
	t.Run("valid", func(t *testing.T) {
		ids, err := resolveCharacterList(state, map[string]any{"characters": []any{"Alice", "Bob"}}, "characters")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 {
			t.Fatalf("want 2, got %d", len(ids))
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := resolveCharacterList(state, map[string]any{"characters": []any{"Charlie"}}, "characters")
		if err == nil {
			t.Fatal("expected error for unknown actor")
		}
	})
}

func TestReadDamageFlagExpect(t *testing.T) {
	t.Run("no_flags", func(t *testing.T) {
		_, ok := readDamageFlagExpect(map[string]any{})
		if ok {
			t.Fatal("expected no flags")
		}
	})
	t.Run("one_flag", func(t *testing.T) {
		expect, ok := readDamageFlagExpect(map[string]any{"resist_physical": true})
		if !ok {
			t.Fatal("expected flags present")
		}
		if expect.resistPhysical == nil || !*expect.resistPhysical {
			t.Fatal("expected resist_physical=true")
		}
	})
	t.Run("all_flags", func(t *testing.T) {
		args := map[string]any{
			"resist_physical": true,
			"resist_magic":    false,
			"immune_physical": true,
			"immune_magic":    false,
		}
		expect, ok := readDamageFlagExpect(args)
		if !ok {
			t.Fatal("expected flags present")
		}
		if expect.resistPhysical == nil || !*expect.resistPhysical {
			t.Fatal("resist_physical")
		}
		if expect.resistMagic == nil || *expect.resistMagic {
			t.Fatal("resist_magic")
		}
		if expect.immunePhysical == nil || !*expect.immunePhysical {
			t.Fatal("immune_physical")
		}
		if expect.immuneMagic == nil || *expect.immuneMagic {
			t.Fatal("immune_magic")
		}
	})
}

func TestMatchesOutcomeHint_Pure(t *testing.T) {
	tests := []struct {
		hint    string
		outcome daggerheartdomain.Outcome
		isCrit  bool
		want    bool
	}{
		{"fear", daggerheartdomain.OutcomeRollWithFear, false, true},
		{"fear", daggerheartdomain.OutcomeRollWithHope, false, false},
		{"hope", daggerheartdomain.OutcomeRollWithHope, false, true},
		{"hope", daggerheartdomain.OutcomeSuccessWithHope, false, true},
		{"hope", daggerheartdomain.OutcomeRollWithFear, false, false},
		{"critical", daggerheartdomain.OutcomeRollWithHope, true, true},
		{"critical", daggerheartdomain.OutcomeRollWithHope, false, false},
		{"failure_hope", daggerheartdomain.OutcomeFailureWithHope, false, true},
		{"failure_hope", daggerheartdomain.OutcomeSuccessWithHope, false, false},
		{"unknown", daggerheartdomain.OutcomeRollWithHope, false, false},
	}
	for _, tc := range tests {
		name := fmt.Sprintf("%s_%v_crit=%v", tc.hint, tc.outcome, tc.isCrit)
		t.Run(name, func(t *testing.T) {
			result := daggerheartdomain.ActionResult{Outcome: tc.outcome, IsCrit: tc.isCrit}
			got := matchesOutcomeHint(result, tc.hint)
			if got != tc.want {
				t.Fatalf("matchesOutcomeHint(%v, %q) = %v, want %v", result, tc.hint, got, tc.want)
			}
		})
	}
}

func TestParseOutcomeBranchSteps(t *testing.T) {
	t.Run("single_map", func(t *testing.T) {
		steps, err := parseOutcomeBranchSteps(map[string]any{"kind": "clear_spotlight"}, "DAGGERHEART")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(steps) != 1 || steps[0].Kind != "clear_spotlight" {
			t.Fatalf("steps = %v", steps)
		}
		if steps[0].System != "" {
			t.Fatalf("system = %q, want empty", steps[0].System)
		}
	})
	t.Run("list_of_maps", func(t *testing.T) {
		steps, err := parseOutcomeBranchSteps([]any{
			map[string]any{"kind": "attack"},
			map[string]any{"kind": "attack", "foo": "bar", "system": "daggerheart"},
		}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(steps) != 2 || steps[0].Kind != "attack" || steps[1].Kind != "attack" {
			t.Fatalf("steps = %v", steps)
		}
		if _, ok := steps[1].Args["foo"]; !ok {
			t.Fatalf("expected args to carry non-kind fields: %#v", steps[1].Args)
		}
		if steps[1].System != "DAGGERHEART" {
			t.Fatalf("system = %q, want DAGGERHEART", steps[1].System)
		}
		if _, ok := steps[1].Args["system"]; ok {
			t.Fatalf("branch step args must not keep system key: %#v", steps[1].Args)
		}
	})
	t.Run("invalid_step", func(t *testing.T) {
		_, err := parseOutcomeBranchSteps([]any{"bad"}, "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveOutcomeBranches(t *testing.T) {
	t.Run("extract_known", func(t *testing.T) {
		branches, err := resolveOutcomeBranches(map[string]any{
			"on_success": []any{map[string]any{"kind": "clear_spotlight"}},
			"on_failure": []any{map[string]any{"kind": "clear_spotlight"}},
		}, map[string]struct{}{
			"on_success": {},
			"on_failure": {},
		}, "DAGGERHEART")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(branches) != 2 {
			t.Fatalf("want 2 branches, got %d", len(branches))
		}
		if branches["on_success"][0].System != "" {
			t.Fatalf("branch system = %q, want empty", branches["on_success"][0].System)
		}
	})
	t.Run("unknown_branch", func(t *testing.T) {
		_, err := resolveOutcomeBranches(map[string]any{
			"on_magic": []any{map[string]any{"kind": "clear_spotlight"}},
		}, map[string]struct{}{
			"on_success": {},
		}, "DAGGERHEART")
		if err == nil {
			t.Fatal("expected unknown branch error")
		}
	})
	t.Run("critical_alias_conflict", func(t *testing.T) {
		_, err := resolveOutcomeBranches(map[string]any{
			"on_critical": []any{map[string]any{"kind": "clear_spotlight"}},
			"on_crit":     []any{map[string]any{"kind": "clear_spotlight"}},
		}, map[string]struct{}{
			"on_critical": {},
			"on_crit":     {},
		}, "DAGGERHEART")
		if err == nil {
			t.Fatal("expected critical alias conflict error")
		}
	})
}

func TestEvaluateActionOutcomeBranch(t *testing.T) {
	tests := []struct {
		name   string
		result actionRollResult
		want   map[string]bool
	}{
		{
			name:   "success_with_hope",
			result: actionRollResult{success: true, hopeDie: 6, fearDie: 2, crit: true},
			want: map[string]bool{
				"on_success":      true,
				"on_failure":      false,
				"on_hope":         true,
				"on_fear":         false,
				"on_success_hope": true,
				"on_failure_hope": false,
				"on_success_fear": false,
				"on_failure_fear": false,
				"on_critical":     true,
				"on_crit":         true,
			},
		},
		{
			name:   "failure_with_fear",
			result: actionRollResult{success: false, hopeDie: 2, fearDie: 6, crit: false},
			want: map[string]bool{
				"on_success":      false,
				"on_failure":      true,
				"on_hope":         false,
				"on_fear":         true,
				"on_success_hope": false,
				"on_failure_hope": false,
				"on_success_fear": false,
				"on_failure_fear": true,
				"on_critical":     false,
				"on_crit":         false,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for branch, want := range tc.want {
				if got := evaluateActionOutcomeBranch(tc.result, branch); got != want {
					t.Fatalf("branch %s got %v want %v", branch, got, want)
				}
			}
		})
	}
}

func TestEvaluateReactionOutcomeBranch(t *testing.T) {
	successWithHope := &daggerheartv1.DaggerheartReactionOutcomeResult{
		Success: true,
		Outcome: daggerheartv1.Outcome_SUCCESS_WITH_HOPE,
		Crit:    true,
	}
	failureWithFear := &daggerheartv1.DaggerheartReactionOutcomeResult{
		Success: false,
		Outcome: daggerheartv1.Outcome_FAILURE_WITH_FEAR,
		Crit:    false,
	}
	if !evaluateReactionOutcomeBranch(successWithHope, "on_success") ||
		!evaluateReactionOutcomeBranch(successWithHope, "on_hope") ||
		!evaluateReactionOutcomeBranch(successWithHope, "on_success_hope") ||
		!evaluateReactionOutcomeBranch(successWithHope, "on_critical") ||
		evaluateReactionOutcomeBranch(successWithHope, "on_failure_fear") ||
		evaluateReactionOutcomeBranch(successWithHope, "on_success_fear") {
		t.Fatal("expected success+hope+critical to match")
	}
	if !evaluateReactionOutcomeBranch(failureWithFear, "on_failure") ||
		!evaluateReactionOutcomeBranch(failureWithFear, "on_fear") ||
		!evaluateReactionOutcomeBranch(failureWithFear, "on_failure_fear") ||
		evaluateReactionOutcomeBranch(failureWithFear, "on_success") ||
		evaluateReactionOutcomeBranch(failureWithFear, "on_success_fear") {
		t.Fatal("expected failure+fear to match")
	}
	if evaluateReactionOutcomeBranch(nil, "on_success") {
		t.Fatal("nil reaction result should not match")
	}
}

func TestActorID(t *testing.T) {
	state := &scenarioState{actors: map[string]string{"Alice": "a-1", "Bob": "b-1"}}

	t.Run("exact", func(t *testing.T) {
		id, err := actorID(state, "Alice")
		if err != nil || id != "a-1" {
			t.Fatalf("want a-1, got %s, err=%v", id, err)
		}
	})
	t.Run("case_insensitive", func(t *testing.T) {
		id, err := actorID(state, "alice")
		if err != nil || id != "a-1" {
			t.Fatalf("want a-1, got %s, err=%v", id, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := actorID(state, "Charlie")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParticipantID(t *testing.T) {
	state := &scenarioState{participants: map[string]string{"Guide": "p-1", "Rhea": "p-2"}}

	t.Run("exact", func(t *testing.T) {
		id, err := participantID(state, "Guide")
		if err != nil || id != "p-1" {
			t.Fatalf("want p-1, got %s, err=%v", id, err)
		}
	})
	t.Run("case_insensitive", func(t *testing.T) {
		id, err := participantID(state, "rHeA")
		if err != nil || id != "p-2" {
			t.Fatalf("want p-2, got %s, err=%v", id, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := participantID(state, "Bryn")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAdversaryID(t *testing.T) {
	state := &scenarioState{adversaries: map[string]string{"Dragon": "d-1"}}

	t.Run("exact", func(t *testing.T) {
		id, err := adversaryID(state, "Dragon")
		if err != nil || id != "d-1" {
			t.Fatalf("want d-1, got %s, err=%v", id, err)
		}
	})
	t.Run("case_insensitive", func(t *testing.T) {
		id, err := adversaryID(state, "dragon")
		if err != nil || id != "d-1" {
			t.Fatalf("want d-1, got %s, err=%v", id, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := adversaryID(state, "Goblin")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveTargetID(t *testing.T) {
	state := &scenarioState{
		actors:      map[string]string{"Alice": "a-1"},
		adversaries: map[string]string{"Dragon": "d-1"},
	}

	t.Run("actor", func(t *testing.T) {
		id, isAdv, err := resolveTargetID(state, "Alice")
		if err != nil || isAdv || id != "a-1" {
			t.Fatalf("want a-1/false, got %s/%v/%v", id, isAdv, err)
		}
	})
	t.Run("adversary", func(t *testing.T) {
		id, isAdv, err := resolveTargetID(state, "Dragon")
		if err != nil || !isAdv || id != "d-1" {
			t.Fatalf("want d-1/true, got %s/%v/%v", id, isAdv, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, _, err := resolveTargetID(state, "Ghost")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAllActorIDs(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{}}
		ids := allActorIDs(state)
		if ids != nil {
			t.Fatalf("expected nil, got %v", ids)
		}
	})
	t.Run("sorted", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{"Bob": "b-1", "Alice": "a-1"}}
		ids := allActorIDs(state)
		if len(ids) != 2 || ids[0] != "a-1" || ids[1] != "b-1" {
			t.Fatalf("expected sorted [a-1,b-1], got %v", ids)
		}
	})
}

func TestIsSessionEvent(t *testing.T) {
	if !isSessionEvent("action.roll") {
		t.Fatal("expected true for action.*")
	}
	if !isSessionEvent("session.started") {
		t.Fatal("expected true for session.*")
	}
	if isSessionEvent("campaign.created") {
		t.Fatal("expected false for campaign.*")
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if uniqueNonEmptyStrings(nil) != nil {
			t.Fatal("expected nil")
		}
	})
	t.Run("dedup_and_trim", func(t *testing.T) {
		got := uniqueNonEmptyStrings([]string{"a", " b ", "a", "", "b"})
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("want [a,b], got %v", got)
		}
	})
}

func TestOptionalString(t *testing.T) {
	args := map[string]any{"key": "value"}
	if optionalString(args, "key", "fb") != "value" {
		t.Fatal("expected value")
	}
	if optionalString(args, "missing", "fb") != "fb" {
		t.Fatal("expected fallback")
	}
	if optionalString(map[string]any{"key": ""}, "key", "fb") != "fb" {
		t.Fatal("expected fallback for empty string")
	}
}

func TestOptionalInt(t *testing.T) {
	if optionalInt(map[string]any{"k": 42}, "k", 0) != 42 {
		t.Fatal("expected 42")
	}
	if optionalInt(map[string]any{"k": float64(42)}, "k", 0) != 42 {
		t.Fatal("expected 42 from float64")
	}
	if optionalInt(map[string]any{}, "k", 99) != 99 {
		t.Fatal("expected fallback 99")
	}
}

func TestOptionalBool(t *testing.T) {
	if !optionalBool(map[string]any{"k": true}, "k", false) {
		t.Fatal("expected true")
	}
	if !optionalBool(map[string]any{"k": "true"}, "k", false) {
		t.Fatal("expected true from string")
	}
	if !optionalBool(map[string]any{"k": "yes"}, "k", false) {
		t.Fatal("expected true from 'yes'")
	}
	if optionalBool(map[string]any{"k": "false"}, "k", true) {
		t.Fatal("expected false from 'false'")
	}
	if !optionalBool(map[string]any{}, "k", true) {
		t.Fatal("expected fallback true")
	}
}

func TestReadInt(t *testing.T) {
	v, ok := readInt(map[string]any{"k": 5}, "k")
	if !ok || v != 5 {
		t.Fatal("expected 5")
	}
	_, ok = readInt(map[string]any{}, "k")
	if ok {
		t.Fatal("expected not ok")
	}
}

func TestReadBool(t *testing.T) {
	v, ok := readBool(map[string]any{"k": true}, "k")
	if !ok || !v {
		t.Fatal("expected true")
	}
	v, ok = readBool(map[string]any{"k": "no"}, "k")
	if !ok || v {
		t.Fatal("expected false from 'no'")
	}
	_, ok = readBool(map[string]any{"k": "maybe"}, "k")
	if ok {
		t.Fatal("expected not ok for 'maybe'")
	}
}

func TestReadStringSlice(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		got := readStringSlice(map[string]any{"k": []any{"a", " b "}}, "k")
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("want [a,b], got %v", got)
		}
	})
	t.Run("missing", func(t *testing.T) {
		got := readStringSlice(map[string]any{}, "k")
		if got != nil {
			t.Fatal("expected nil")
		}
	})
	t.Run("empty_strings_filtered", func(t *testing.T) {
		got := readStringSlice(map[string]any{"k": []any{"", " "}}, "k")
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})
	t.Run("go_string_slice", func(t *testing.T) {
		got := readStringSlice(map[string]any{"k": []string{"a", " b ", ""}}, "k")
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("want [a,b], got %v", got)
		}
	})
}

func TestNormalizeModifierSource(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Experience", "experience"},
		{"Tag Team", "tag_team"},
		{"hope-feature", "hope_feature"},
		{" ", ""},
		{"", ""},
	}
	for _, tc := range tests {
		if got := normalizeModifierSource(tc.in); got != tc.want {
			t.Fatalf("normalizeModifierSource(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsHopeSpendSource(t *testing.T) {
	for _, src := range []string{"experience", "help", "Tag Team", "hope-feature"} {
		if !isHopeSpendSource(src) {
			t.Fatalf("expected true for %q", src)
		}
	}
	if isHopeSpendSource("random") {
		t.Fatal("expected false for random")
	}
}

func TestParseDamageType_Variants(t *testing.T) {
	if parseDamageType("MAGIC") != daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC {
		t.Fatal("expected MAGIC")
	}
	if parseDamageType("  mixed  ") != daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED {
		t.Fatal("expected MIXED")
	}
}

func TestParseGameSystem(t *testing.T) {
	gs, err := parseGameSystem("DAGGERHEART")
	if err != nil || gs != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("want DAGGERHEART, got %v, err=%v", gs, err)
	}
	gs, err = parseGameSystem("GAME_SYSTEM_DAGGERHEART")
	if err != nil || gs != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("want GAME_SYSTEM_DAGGERHEART, got %v, err=%v", gs, err)
	}
	_, err = parseGameSystem("GAME_SYSTEM_UNSPECIFIED")
	if err == nil {
		t.Fatal("expected error for unspecified")
	}
	_, err = parseGameSystem("unknown")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGmMode(t *testing.T) {
	gm, err := parseGmMode("HUMAN")
	if err != nil || gm != gamev1.GmMode_HUMAN {
		t.Fatal("expected HUMAN")
	}
	gm, err = parseGmMode("AI")
	if err != nil || gm != gamev1.GmMode_AI {
		t.Fatal("expected AI")
	}
	gm, err = parseGmMode("HYBRID")
	if err != nil || gm != gamev1.GmMode_HYBRID {
		t.Fatal("expected HYBRID")
	}
	_, err = parseGmMode("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCharacterKind(t *testing.T) {
	ck, err := parseCharacterKind("PC")
	if err != nil || ck != gamev1.CharacterKind_PC {
		t.Fatal("expected PC")
	}
	ck, err = parseCharacterKind("NPC")
	if err != nil || ck != gamev1.CharacterKind_NPC {
		t.Fatal("expected NPC")
	}
	_, err = parseCharacterKind("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseParticipantRole(t *testing.T) {
	r, err := parseParticipantRole("PLAYER")
	if err != nil || r != gamev1.ParticipantRole_PLAYER {
		t.Fatal("expected PLAYER")
	}
	r, err = parseParticipantRole("")
	if err != nil || r != gamev1.ParticipantRole_PLAYER {
		t.Fatal("expected PLAYER for empty")
	}
	r, err = parseParticipantRole("GM")
	if err != nil || r != gamev1.ParticipantRole_GM {
		t.Fatal("expected GM")
	}
	_, err = parseParticipantRole("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseController(t *testing.T) {
	c, err := parseController("HUMAN")
	if err != nil || c != gamev1.Controller_CONTROLLER_HUMAN {
		t.Fatal("expected HUMAN")
	}
	c, err = parseController("")
	if err != nil || c != gamev1.Controller_CONTROLLER_HUMAN {
		t.Fatal("expected HUMAN for empty")
	}
	c, err = parseController("AI")
	if err != nil || c != gamev1.Controller_CONTROLLER_AI {
		t.Fatal("expected AI")
	}
	_, err = parseController("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseControl(t *testing.T) {
	for _, val := range []string{"participant", "gm", "none"} {
		got, err := parseControl(val)
		if err != nil || got != val {
			t.Fatalf("parseControl(%q) = %q, err=%v", val, got, err)
		}
	}
	got, err := parseControl("")
	if err != nil || got != "" {
		t.Fatal("expected empty for empty")
	}
	_, err = parseControl("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseLifeState(t *testing.T) {
	tests := []struct {
		input string
		want  daggerheartv1.DaggerheartLifeState
	}{
		{"alive", daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE},
		{"unconscious", daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS},
		{"blaze_of_glory", daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY},
		{"dead", daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD},
	}
	for _, tc := range tests {
		got, err := parseLifeState(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("parseLifeState(%q) = %v, err=%v", tc.input, got, err)
		}
	}
	_, err := parseLifeState("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRestType(t *testing.T) {
	rt, err := parseRestType("short")
	if err != nil || rt != daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT {
		t.Fatal("expected SHORT")
	}
	rt, err = parseRestType("long")
	if err != nil || rt != daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG {
		t.Fatal("expected LONG")
	}
	_, err = parseRestType("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownKind(t *testing.T) {
	ck, err := parseCountdownKind("progress")
	if err != nil || ck != daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS {
		t.Fatal("expected PROGRESS")
	}
	ck, err = parseCountdownKind("consequence")
	if err != nil || ck != daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE {
		t.Fatal("expected CONSEQUENCE")
	}
	_, err = parseCountdownKind("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownAdvancementPolicy(t *testing.T) {
	policy, err := parseCountdownAdvancementPolicy("manual")
	if err != nil || policy != daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL {
		t.Fatal("expected MANUAL")
	}
	policy, err = parseCountdownAdvancementPolicy("long_rest")
	if err != nil || policy != daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST {
		t.Fatal("expected LONG_REST")
	}
	_, err = parseCountdownAdvancementPolicy("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownDirection(t *testing.T) {
	cd, err := parseCountdownDirection("increase")
	if err != nil || cd != daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE {
		t.Fatal("expected INCREASE")
	}
	cd, err = parseCountdownDirection("decrease")
	if err != nil || cd != daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START {
		t.Fatal("expected DECREASE")
	}
	_, err = parseCountdownDirection("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownLoopBehavior(t *testing.T) {
	loop, err := parseCountdownLoopBehavior("reset")
	if err != nil || loop != daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET {
		t.Fatal("expected RESET")
	}
	loop, err = parseCountdownLoopBehavior("reset_increase_start")
	if err != nil || loop != daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START {
		t.Fatal("expected RESET_INCREASE_START")
	}
	_, err = parseCountdownLoopBehavior("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownStatus(t *testing.T) {
	statusValue, err := parseCountdownStatus("active")
	if err != nil || statusValue != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE {
		t.Fatal("expected ACTIVE")
	}
	statusValue, err = parseCountdownStatus("trigger_pending")
	if err != nil || statusValue != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING {
		t.Fatal("expected TRIGGER_PENDING")
	}
	_, err = parseCountdownStatus("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDowntimeMove(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"clear_all_stress", "clear_all_stress"},
		{"repair_all_armor", "repair_all_armor"},
		{"prepare", "prepare"},
		{"work_on_project", "work_on_project"},
	}
	for _, tc := range tests {
		got, err := parseDowntimeMove(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("parseDowntimeMove(%q) = %v, err=%v", tc.input, got, err)
		}
	}
	_, err := parseDowntimeMove("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTemporaryArmorDuration(t *testing.T) {
	tests := []string{"short_rest", "long_rest", "session", "scene"}
	for _, input := range tests {
		got, err := parseTemporaryArmorDuration(input)
		if err != nil || got != input {
			t.Fatalf("parseTemporaryArmorDuration(%q) = %q, err=%v", input, got, err)
		}
	}
	_, err := parseTemporaryArmorDuration("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDeathMove(t *testing.T) {
	tests := []struct {
		input string
		want  daggerheartv1.DaggerheartDeathMove
	}{
		{"blaze_of_glory", daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY},
		{"avoid_death", daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH},
		{"risk_it_all", daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL},
	}
	for _, tc := range tests {
		got, err := parseDeathMove(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("parseDeathMove(%q) = %v, err=%v", tc.input, got, err)
		}
	}
	_, err := parseDeathMove("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseConditions(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		conds, err := parseConditions([]string{"VULNERABLE", "RESTRAINED", "HIDDEN"})
		if err != nil || len(conds) != 3 {
			t.Fatalf("expected 3 conditions, got %v, err=%v", conds, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := parseConditions([]string{"UNKNOWN"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBuildDamageDice(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{})
		if len(dice) != 1 || dice[0].Sides != 6 || dice[0].Count != 1 {
			t.Fatalf("expected 1d6, got %v", dice)
		}
	})
	t.Run("custom", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{
			"damage_dice": []any{
				map[string]any{"sides": 8, "count": 2},
			},
		})
		if len(dice) != 1 || dice[0].Sides != 8 || dice[0].Count != 2 {
			t.Fatalf("expected 2d8, got %v", dice)
		}
	})
	t.Run("empty_list_falls_back", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{"damage_dice": []any{}})
		if len(dice) != 1 || dice[0].Sides != 6 {
			t.Fatalf("expected default 1d6, got %v", dice)
		}
	})
}

func TestBuildActionRollModifiers(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		mods := buildActionRollModifiers(map[string]any{}, "mods")
		if mods != nil {
			t.Fatal("expected nil")
		}
	})
	t.Run("with_values", func(t *testing.T) {
		args := map[string]any{
			"mods": []any{
				map[string]any{"source": "buff", "value": 2},
				map[string]any{"source": "experience"},
			},
		}
		mods := buildActionRollModifiers(args, "mods")
		if len(mods) != 2 {
			t.Fatalf("expected 2 modifiers, got %d", len(mods))
		}
		if mods[0].Source != "buff" || mods[0].Value != 2 {
			t.Fatalf("mod 0: %+v", mods[0])
		}
		// experience is a hope spend source, so value=0 is ok
		if mods[1].Source != "experience" || mods[1].Value != 0 {
			t.Fatalf("mod 1: %+v", mods[1])
		}
	})
	t.Run("with_flat_modifier", func(t *testing.T) {
		mods := buildActionRollModifiers(map[string]any{
			"mods":     []any{map[string]any{"source": "buff", "value": 2}},
			"modifier": 3,
		}, "mods")
		if len(mods) != 2 {
			t.Fatalf("expected 2 modifiers, got %d", len(mods))
		}
		if mods[1].Source != "modifier" || mods[1].Value != 3 {
			t.Fatalf("flat modifier not forwarded: %+v", mods[1])
		}
	})
}

func TestBuildAdversaryRollModifiers(t *testing.T) {
	t.Run("uses_explicit_modifiers_first", func(t *testing.T) {
		mods := buildAdversaryRollModifiers(map[string]any{
			"modifiers": []any{map[string]any{"source": "experience", "value": 2}},
			"modifier":  1,
		})
		if len(mods) != 2 {
			t.Fatalf("expected 2 modifiers, got %d", len(mods))
		}
		if mods[0].Source != "experience" || mods[0].Value != 2 {
			t.Fatalf("mod 0 = %+v", mods[0])
		}
		if mods[1].Source != "modifier" || mods[1].Value != 1 {
			t.Fatalf("mod 1 = %+v", mods[1])
		}
	})
	t.Run("falls_back_to_attack_modifier", func(t *testing.T) {
		mods := buildAdversaryRollModifiers(map[string]any{"attack_modifier": 3})
		if len(mods) != 1 {
			t.Fatalf("expected 1 modifier, got %d", len(mods))
		}
		if mods[0].Source != "attack_modifier" || mods[0].Value != 3 {
			t.Fatalf("modifier = %+v", mods[0])
		}
	})
}

func TestResolveCountdownID(t *testing.T) {
	state := &scenarioState{countdowns: map[string]string{"timer": "cd-1"}}

	t.Run("by_id", func(t *testing.T) {
		id, err := resolveCountdownID(state, map[string]any{"countdown_id": "cd-direct"})
		if err != nil || id != "cd-direct" {
			t.Fatalf("want cd-direct, got %s, err=%v", id, err)
		}
	})
	t.Run("by_name", func(t *testing.T) {
		id, err := resolveCountdownID(state, map[string]any{"name": "timer"})
		if err != nil || id != "cd-1" {
			t.Fatalf("want cd-1, got %s, err=%v", id, err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := resolveCountdownID(state, map[string]any{"name": "unknown"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("empty", func(t *testing.T) {
		id, err := resolveCountdownID(state, map[string]any{})
		if err != nil || id != "" {
			t.Fatalf("want empty, got %q, err=%v", id, err)
		}
	})
}

func TestPrefabOptions(t *testing.T) {
	t.Run("frodo", func(t *testing.T) {
		opts := prefabOptions("frodo")
		if opts["kind"] != "PC" || opts["hp_max"] != 6 {
			t.Fatalf("unexpected frodo options: %v", opts)
		}
	})
	t.Run("default", func(t *testing.T) {
		opts := prefabOptions("unknown")
		if opts["kind"] != "PC" {
			t.Fatalf("expected default PC kind, got %v", opts)
		}
		if len(opts) != 1 {
			t.Fatalf("expected only kind field, got %v", opts)
		}
	})
}

func TestRequiredString(t *testing.T) {
	if requiredString(map[string]any{"k": "v"}, "k") != "v" {
		t.Fatal("expected v")
	}
	if requiredString(map[string]any{}, "k") != "" {
		t.Fatal("expected empty")
	}
	if requiredString(map[string]any{"k": ""}, "k") != "" {
		t.Fatal("expected empty for empty string")
	}
}

func TestRequireDamageDice(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		if err := requireDamageDice(map[string]any{}, "ctx"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("empty_list", func(t *testing.T) {
		if err := requireDamageDice(map[string]any{"damage_dice": []any{}}, "ctx"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		if err := requireDamageDice(map[string]any{"damage_dice": []any{map[string]any{"sides": 6}}}, "ctx"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Method tests — require Runner with fakes
// ---------------------------------------------------------------------------

func TestEnsureSession_NoCampaign(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{}
	if err := r.ensureSession(context.Background(), state); err == nil {
		t.Fatal("expected error for missing campaign")
	}
}

func TestEnsureSession_AlreadySet(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "c-1", sessionID: "s-1"}
	if err := r.ensureSession(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureSession_AutoCreates(t *testing.T) {
	sess := &fakeSessionClient{
		startSession: func(_ context.Context, in *gamev1.StartSessionRequest, _ ...grpc.CallOption) (*gamev1.StartSessionResponse, error) {
			return &gamev1.StartSessionResponse{
				Session: &gamev1.Session{Id: "s-new"},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{sessionClient: sess})
	state := &scenarioState{campaignID: "c-1"}

	if err := r.ensureSession(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.sessionID != "s-new" {
		t.Fatalf("expected s-new, got %s", state.sessionID)
	}
}

func TestEnsureSession_NilSession(t *testing.T) {
	sess := &fakeSessionClient{
		startSession: func(context.Context, *gamev1.StartSessionRequest, ...grpc.CallOption) (*gamev1.StartSessionResponse, error) {
			return &gamev1.StartSessionResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{sessionClient: sess})
	state := &scenarioState{campaignID: "c-1"}

	if err := r.ensureSession(context.Background(), state); err == nil {
		t.Fatal("expected error for nil session")
	}
}

func TestEnsureCampaign(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	t.Run("empty", func(t *testing.T) {
		if err := r.ensureCampaign(&scenarioState{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("present", func(t *testing.T) {
		if err := r.ensureCampaign(&scenarioState{campaignID: "c-1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLatestSeq(t *testing.T) {
	t.Run("empty_campaign", func(t *testing.T) {
		r := newTestRunner(scenarioEnv{})
		seq, err := r.latestSeq(context.Background(), &scenarioState{})
		if err != nil || seq != 0 {
			t.Fatalf("want 0, got %d, err=%v", seq, err)
		}
	})
	t.Run("returns_seq", func(t *testing.T) {
		evt := &fakeEventClient{seq: 41}
		r := newTestRunner(scenarioEnv{eventClient: evt})
		seq, err := r.latestSeq(context.Background(), &scenarioState{campaignID: "c-1"})
		if err != nil || seq != 42 {
			t.Fatalf("want 42, got %d, err=%v", seq, err)
		}
	})
}

func TestGetSnapshot_HappyPath(t *testing.T) {
	snap := &fakeSnapshotClient{
		getSnapshot: func(context.Context, *gamev1.GetSnapshotRequest, ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return &gamev1.GetSnapshotResponse{
				Snapshot: &gamev1.Snapshot{
					SystemSnapshot: &gamev1.Snapshot_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 3},
					},
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{snapshotClient: snap})
	state := &scenarioState{campaignID: "c-1"}

	got, err := r.getSnapshot(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GmFear != 3 {
		t.Fatalf("want gm_fear=3, got %d", got.GmFear)
	}
}

func TestGetSnapshot_Error(t *testing.T) {
	snap := &fakeSnapshotClient{
		getSnapshot: func(context.Context, *gamev1.GetSnapshotRequest, ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return nil, fmt.Errorf("snapshot error")
		},
	}
	r := newTestRunner(scenarioEnv{snapshotClient: snap})

	_, err := r.getSnapshot(context.Background(), &scenarioState{campaignID: "c-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSnapshot_NilResponse(t *testing.T) {
	snap := &fakeSnapshotClient{
		getSnapshot: func(context.Context, *gamev1.GetSnapshotRequest, ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return &gamev1.GetSnapshotResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{snapshotClient: snap})

	_, err := r.getSnapshot(context.Background(), &scenarioState{campaignID: "c-1"})
	if err == nil {
		t.Fatal("expected error for nil snapshot")
	}
}

func TestGetCharacterState_HappyPath(t *testing.T) {
	char := &fakeCharacterClient{
		getSheet: func(_ context.Context, in *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10},
					},
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char})
	state := &scenarioState{campaignID: "c-1"}

	cs, err := r.getCharacterState(context.Background(), state, "char-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Hp != 10 {
		t.Fatalf("want hp=10, got %d", cs.Hp)
	}
}

func TestGetCharacterState_NilState(t *testing.T) {
	char := &fakeCharacterClient{
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char})

	_, err := r.getCharacterState(context.Background(), &scenarioState{campaignID: "c-1"}, "char-1")
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestGetAdversary_HappyPath(t *testing.T) {
	dh := &fakeDaggerheartClient{
		getAdversary: func(context.Context, *daggerheartv1.DaggerheartGetAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartGetAdversaryResponse, error) {
			return &daggerheartv1.DaggerheartGetAdversaryResponse{
				Adversary: &daggerheartv1.DaggerheartAdversary{Hp: 20},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{daggerheartClient: dh})

	adv, err := r.getAdversary(context.Background(), &scenarioState{campaignID: "c-1"}, "adv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv.Hp != 20 {
		t.Fatalf("want hp=20, got %d", adv.Hp)
	}
}

func TestGetAdversary_NilAdversary(t *testing.T) {
	dh := &fakeDaggerheartClient{
		getAdversary: func(context.Context, *daggerheartv1.DaggerheartGetAdversaryRequest, ...grpc.CallOption) (*daggerheartv1.DaggerheartGetAdversaryResponse, error) {
			return &daggerheartv1.DaggerheartGetAdversaryResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{daggerheartClient: dh})

	_, err := r.getAdversary(context.Background(), &scenarioState{campaignID: "c-1"}, "adv-1")
	if err == nil {
		t.Fatal("expected error for nil adversary")
	}
}

func TestApplyDefaultDaggerheartProfile(t *testing.T) {
	var patchedProfile *daggerheartv1.DaggerheartProfile
	baseProfile := &daggerheartv1.DaggerheartProfile{
		Level:           2,
		HpMax:           7,
		StressMax:       wrapperspb.Int32(6),
		Evasion:         wrapperspb.Int32(11),
		MajorThreshold:  wrapperspb.Int32(6),
		SevereThreshold: wrapperspb.Int32(12),
		ArmorMax:        wrapperspb.Int32(1),
		ArmorScore:      wrapperspb.Int32(1),
	}
	char := &fakeCharacterClient{
		getSheet: func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				Profile: &gamev1.CharacterProfile{
					SystemProfile: &gamev1.CharacterProfile_Daggerheart{Daggerheart: baseProfile},
				},
			}, nil
		},
		patchProfile: func(_ context.Context, in *gamev1.PatchCharacterProfileRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			patchedProfile = in.GetDaggerheart()
			return &gamev1.PatchCharacterProfileResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char})
	state := &scenarioState{campaignID: "c-1"}

	t.Run("defaults", func(t *testing.T) {
		patchedProfile = nil
		args := map[string]any{"name": "Frodo", "kind": "PC"}
		_, err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile != nil {
			t.Fatalf("expected no default profile patch")
		}
	})
	t.Run("custom_overrides", func(t *testing.T) {
		patchedProfile = nil
		args := map[string]any{"level": 5, "hp_max": 20, "armor_max": 3}
		_, err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile == nil {
			t.Fatal("expected profile patch")
		}
		if patchedProfile.Level != 5 {
			t.Fatalf("expected level=5, got %d", patchedProfile.Level)
		}
		if patchedProfile.HpMax != 20 {
			t.Fatalf("expected hp_max=20, got %d", patchedProfile.HpMax)
		}
		if patchedProfile.ArmorMax.GetValue() != 3 {
			t.Fatalf("expected armor_max=3, got %d", patchedProfile.ArmorMax.GetValue())
		}
	})
	t.Run("armor_max_overrides_armor", func(t *testing.T) {
		patchedProfile = nil
		args := map[string]any{"armor": 3, "armor_max": 5}
		_, err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile == nil {
			t.Fatal("expected profile patch")
		}
		if patchedProfile.ArmorMax.GetValue() != 5 {
			t.Fatalf("expected armor_max=5, got %d", patchedProfile.ArmorMax.GetValue())
		}
	})
	t.Run("armor_only_skips_profile_patch", func(t *testing.T) {
		patchedProfile = nil
		args := map[string]any{"armor": 3}
		_, err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile == nil {
			t.Fatal("expected profile patch for armor-only override")
		}
		if patchedProfile.ArmorMax.GetValue() != 3 {
			t.Fatalf("expected armor_max=3, got %d", patchedProfile.ArmorMax.GetValue())
		}
	})
}

func TestEnsureDaggerheartCharacterReadiness_UsesCatalogBackedEquipmentIDs(t *testing.T) {
	var applyReq *gamev1.ApplyCharacterCreationWorkflowRequest
	char := &fakeCharacterClient{
		applyWorkflow: func(_ context.Context, in *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
			applyReq = in
			return &gamev1.ApplyCharacterCreationWorkflowResponse{}, nil
		},
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				Profile: &gamev1.CharacterProfile{
					SystemProfile: &gamev1.CharacterProfile_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartProfile{
							ClassId:    scenarioReadinessClassID,
							SubclassId: scenarioReadinessSubclassID,
						},
					},
				},
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{
							Hp:      6,
							HopeMax: 2,
						},
					},
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char})
	state := &scenarioState{campaignID: "c-1"}

	err := r.ensureDaggerheartCharacterReadiness(context.Background(), state, "char-1", map[string]any{
		"equipment": map[string]any{
			"armor_id":       "armor.chainmail-armor",
			"potion_item_id": "item.minor-stamina-potion",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applyReq == nil {
		t.Fatal("expected ApplyCharacterCreationWorkflow request")
	}
	input := applyReq.GetDaggerheart()
	if input == nil {
		t.Fatal("expected daggerheart workflow input")
	}
	if got := input.GetEquipmentInput().GetArmorId(); got != "armor.chainmail-armor" {
		t.Fatalf("armor_id = %q, want %q", got, "armor.chainmail-armor")
	}
	if got := input.GetEquipmentInput().GetWeaponIds(); len(got) != 1 || got[0] != scenarioReadinessWeaponID {
		t.Fatalf("weapon_ids = %v, want [%q]", got, scenarioReadinessWeaponID)
	}
	if got := input.GetEquipmentInput().GetPotionItemId(); got != "item.minor-stamina-potion" {
		t.Fatalf("potion_item_id = %q, want %q", got, "item.minor-stamina-potion")
	}
}

func TestEnsureDaggerheartCharacterReadinessWaitsForProjectedSheet(t *testing.T) {
	calls := 0
	char := &fakeCharacterClient{
		applyWorkflow: func(_ context.Context, _ *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
			return &gamev1.ApplyCharacterCreationWorkflowResponse{}, nil
		},
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			calls++
			if calls == 1 {
				return &gamev1.GetCharacterSheetResponse{
					Profile: &gamev1.CharacterProfile{
						SystemProfile: &gamev1.CharacterProfile_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartProfile{},
						},
					},
					State: &gamev1.CharacterState{
						SystemState: &gamev1.CharacterState_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartCharacterState{},
						},
					},
				}, nil
			}
			return &gamev1.GetCharacterSheetResponse{
				Profile: &gamev1.CharacterProfile{
					SystemProfile: &gamev1.CharacterProfile_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartProfile{
							ClassId:    scenarioReadinessClassID,
							SubclassId: scenarioReadinessSubclassID,
						},
					},
				},
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{
							Hp:      6,
							HopeMax: 2,
						},
					},
				},
			}, nil
		},
	}

	r := newTestRunner(scenarioEnv{characterClient: char})
	state := &scenarioState{campaignID: "c-1"}
	if err := r.ensureDaggerheartCharacterReadiness(context.Background(), state, "char-1", nil); err != nil {
		t.Fatalf("ensureDaggerheartCharacterReadiness: %v", err)
	}
	if calls < 2 {
		t.Fatalf("getSheet calls = %d, want at least 2", calls)
	}
}

func TestApplyOptionalCharacterState_NoOp(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "c-1"}

	_, err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyOptionalCharacterState_MergesHP(t *testing.T) {
	var patchedState *daggerheartv1.DaggerheartCharacterState
	char := &fakeCharacterClient{
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{
							Hp:        10,
							Hope:      3,
							HopeMax:   6,
							Stress:    2,
							Armor:     1,
							LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
						},
					},
				},
			}, nil
		},
	}
	snap := &fakeSnapshotClient{
		patchState: func(_ context.Context, in *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
			patchedState = in.GetDaggerheart()
			return &gamev1.PatchCharacterStateResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char, snapshotClient: snap})
	state := &scenarioState{campaignID: "c-1"}

	_, err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"hp": 5}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patchedState.Hp != 5 {
		t.Fatalf("expected hp=5, got %d", patchedState.Hp)
	}
	// Merged from current
	if patchedState.Stress != 2 {
		t.Fatalf("expected stress=2 (merged), got %d", patchedState.Stress)
	}
}

func TestApplyOptionalCharacterState_AppliesHopeOverrides(t *testing.T) {
	var patchedState *daggerheartv1.DaggerheartCharacterState
	char := &fakeCharacterClient{
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{
							Hp:        10,
							Hope:      3,
							HopeMax:   6,
							Stress:    2,
							Armor:     1,
							LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
						},
					},
				},
			}, nil
		},
	}
	snap := &fakeSnapshotClient{
		patchState: func(_ context.Context, in *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
			patchedState = in.GetDaggerheart()
			return &gamev1.PatchCharacterStateResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char, snapshotClient: snap})
	state := &scenarioState{campaignID: "c-1"}

	_, err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"hope": 1, "hope_max": 1}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patchedState.Hope != 1 {
		t.Fatalf("expected hope=1, got %d", patchedState.Hope)
	}
	if patchedState.HopeMax != 1 {
		t.Fatalf("expected hope_max=1, got %d", patchedState.HopeMax)
	}
	if patchedState.Hp != 10 {
		t.Fatalf("expected hp=10 (merged), got %d", patchedState.Hp)
	}
}

func TestApplyOptionalCharacterState_ClampsToProfileCaps(t *testing.T) {
	var patchedState *daggerheartv1.DaggerheartCharacterState
	char := &fakeCharacterClient{
		getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{
							Hp:        7,
							Hope:      3,
							HopeMax:   2,
							Stress:    7,
							Armor:     2,
							LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
						},
					},
				},
			}, nil
		},
	}
	snap := &fakeSnapshotClient{
		patchState: func(_ context.Context, in *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
			patchedState = in.GetDaggerheart()
			return &gamev1.PatchCharacterStateResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char, snapshotClient: snap})
	state := &scenarioState{campaignID: "c-1"}

	_, err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"hope": 2}, &daggerheartv1.DaggerheartProfile{
		HpMax:     6,
		StressMax: wrapperspb.Int32(6),
		ArmorMax:  wrapperspb.Int32(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patchedState == nil {
		t.Fatal("expected patched state")
	}
	if patchedState.GetHp() != 6 {
		t.Fatalf("hp = %d, want 6", patchedState.GetHp())
	}
	if patchedState.GetStress() != 6 {
		t.Fatalf("stress = %d, want 6", patchedState.GetStress())
	}
	if patchedState.GetArmor() != 1 {
		t.Fatalf("armor = %d, want 1", patchedState.GetArmor())
	}
	if patchedState.GetHope() != 2 {
		t.Fatalf("hope = %d, want 2", patchedState.GetHope())
	}
}

func TestApplyOptionalCharacterState_InvalidLifeState(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "c-1"}

	_, err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"life_state": "bad"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid life_state")
	}
}

func TestAssertExpectedSpotlight_Empty(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	err := r.assertExpectedSpotlight(context.Background(), &scenarioState{}, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// captureExpectedDeltas / assertExpectedDeltas tests
// ---------------------------------------------------------------------------

func TestCaptureExpectedDeltas_NoDeltas(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	spec, before, err := r.captureExpectedDeltas(context.Background(), &scenarioState{}, map[string]any{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec != nil || before != nil {
		t.Fatal("expected nil spec and before when no deltas requested")
	}
}

func TestCaptureExpectedDeltas_HopeDelta(t *testing.T) {
	charClient := &fakeCharacterClient{
		getSheet: func(_ context.Context, in *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hope: 3, Stress: 1},
					},
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: charClient})
	state := &scenarioState{
		campaignID: "camp-1",
		actors:     map[string]string{"alice": "ch-1"},
	}
	args := map[string]any{"expect_hope_delta": 1, "expect_target": "alice"}
	spec, before, err := r.captureExpectedDeltas(context.Background(), state, args, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec == nil || before == nil {
		t.Fatal("expected non-nil spec and before")
	}
	if spec.hopeDelta == nil || *spec.hopeDelta != 1 {
		t.Fatalf("expected hope delta 1, got %v", spec.hopeDelta)
	}
	if spec.stressDelta != nil {
		t.Fatal("expected nil stress delta")
	}
	if before.GetHope() != 3 {
		t.Fatalf("expected before hope 3, got %d", before.GetHope())
	}
}

func TestCaptureExpectedDeltas_MissingTarget(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	// Has a delta but no target name — should error.
	args := map[string]any{"expect_hope_delta": 1}
	_, _, err := r.captureExpectedDeltas(context.Background(), &scenarioState{}, args, "")
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestAssertExpectedDeltas_NilSpec(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	err := r.assertExpectedDeltas(context.Background(), &scenarioState{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertExpectedDeltas_MatchingDeltas(t *testing.T) {
	charClient := &fakeCharacterClient{
		getSheet: func(_ context.Context, in *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hope: 4, Stress: 2},
					},
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: charClient})
	state := &scenarioState{
		campaignID: "camp-1",
		actors:     map[string]string{"alice": "ch-1"},
	}
	hopeDelta := 1
	stressDelta := 1
	spec := &expectedDeltas{
		name:        "alice",
		characterID: "ch-1",
		hopeDelta:   &hopeDelta,
		stressDelta: &stressDelta,
	}
	before := &daggerheartv1.DaggerheartCharacterState{Hope: 3, Stress: 1}
	err := r.assertExpectedDeltas(context.Background(), state, spec, before)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// assertExpectedSpotlight extended tests
// ---------------------------------------------------------------------------

func TestAssertExpectedSpotlight_NoneWithNoSpotlight(t *testing.T) {
	sessionClient := &fakeSessionClient{
		getSpotlight: func(context.Context, *gamev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.GetSessionSpotlightResponse, error) {
			return nil, fmt.Errorf("no spotlight")
		},
	}
	r := newTestRunner(scenarioEnv{sessionClient: sessionClient})
	state := &scenarioState{campaignID: "camp-1", sessionID: "s-1"}
	err := r.assertExpectedSpotlight(context.Background(), state, map[string]any{"expect_spotlight": "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertExpectedSpotlight_GM(t *testing.T) {
	sessionClient := &fakeSessionClient{
		getSpotlight: func(context.Context, *gamev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.GetSessionSpotlightResponse, error) {
			return &gamev1.GetSessionSpotlightResponse{
				Spotlight: &gamev1.SessionSpotlight{
					Type: gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM,
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{sessionClient: sessionClient})
	state := &scenarioState{campaignID: "camp-1", sessionID: "s-1"}
	err := r.assertExpectedSpotlight(context.Background(), state, map[string]any{"expect_spotlight": "gm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertExpectedSpotlight_Character(t *testing.T) {
	sessionClient := &fakeSessionClient{
		getSpotlight: func(context.Context, *gamev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*gamev1.GetSessionSpotlightResponse, error) {
			return &gamev1.GetSessionSpotlightResponse{
				Spotlight: &gamev1.SessionSpotlight{
					Type:        gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
					CharacterId: "ch-1",
				},
			}, nil
		},
	}
	r := newTestRunner(scenarioEnv{sessionClient: sessionClient})
	state := &scenarioState{
		campaignID: "camp-1",
		sessionID:  "s-1",
		actors:     map[string]string{"alice": "ch-1"},
	}
	err := r.assertExpectedSpotlight(context.Background(), state, map[string]any{"expect_spotlight": "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertExpectedSpotlight_NoSession(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "camp-1"}
	err := r.assertExpectedSpotlight(context.Background(), state, map[string]any{"expect_spotlight": "gm"})
	if err == nil {
		t.Fatal("expected error for missing session")
	}
}

// ---------------------------------------------------------------------------
// requireEventTypesAfterSeq / requireAnyEventTypesAfterSeq tests
// ---------------------------------------------------------------------------

func TestRequireEventTypesAfterSeq_Found(t *testing.T) {
	eventClient := &fakeEventClient{seq: 10}
	r := newTestRunner(scenarioEnv{eventClient: eventClient})
	state := &scenarioState{campaignID: "camp-1"}
	err := r.requireEventTypesAfterSeq(context.Background(), state, 5, "test.event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireAnyEventTypesAfterSeq_FirstMatch(t *testing.T) {
	eventClient := &fakeEventClient{seq: 10}
	r := newTestRunner(scenarioEnv{eventClient: eventClient})
	state := &scenarioState{campaignID: "camp-1"}
	err := r.requireAnyEventTypesAfterSeq(context.Background(), state, 5, "test.event1", "test.event2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// resolveOpenSessionGate tests
// ---------------------------------------------------------------------------

func TestResolveOpenSessionGate_NoGateEvent(t *testing.T) {
	// fakeEventClient returns events with no payload — resolveOpenSessionGate
	// should fail because no gate_id is found.
	// Same limitation: fakeEventClient always returns events with incrementing seq
	// but no PayloadJson, and gate_id will be empty, so the function will skip
	// all events and return "session gate opened event not found".
	eventClient := &fakeEventClient{seq: 0}
	r := newTestRunner(scenarioEnv{eventClient: eventClient})
	state := &scenarioState{campaignID: "camp-1", sessionID: "s-1"}
	err := r.resolveOpenSessionGate(context.Background(), state, 0)
	if err == nil {
		t.Fatal("expected error for missing gate event")
	}
}

// ---------------------------------------------------------------------------
// assertDamageFlags tests
// ---------------------------------------------------------------------------

func TestAssertDamageFlags_NoExpectations(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	err := r.assertDamageFlags(context.Background(), &scenarioState{}, 0, "", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertDamageFlagsMatchesLatestDamageEvent(t *testing.T) {
	payloadJSON, _ := json.Marshal(daggerheartpayload.DamageAppliedPayload{
		CharacterID:    ids.CharacterID("char-1"),
		ResistPhysical: true,
		ResistMagic:    false,
		ImmunePhysical: false,
		ImmuneMagic:    true,
	})
	eventClient := &fakeEventClient{
		listEvents: func(_ context.Context, req *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
			if !strings.Contains(req.GetFilter(), string(daggerheartpayload.EventTypeDamageApplied)) {
				t.Fatalf("filter = %q, want damage_applied filter", req.GetFilter())
			}
			return &gamev1.ListEventsResponse{
				Events: []*gamev1.Event{
					{Seq: 5, PayloadJson: payloadJSON},
				},
			}, nil
		},
	}

	r := newTestRunner(scenarioEnv{eventClient: eventClient})
	state := &scenarioState{campaignID: "camp-1", sessionID: "session-1", ownerParticipantID: "owner-1"}
	err := r.assertDamageFlags(context.Background(), state, 3, "char-1", map[string]any{
		"resist_physical": true,
		"resist_magic":    false,
		"immune_physical": false,
		"immune_magic":    true,
	})
	if err != nil {
		t.Fatalf("assertDamageFlags: %v", err)
	}
}

func TestAssertDamageFlagsRejectsMismatch(t *testing.T) {
	payloadJSON, _ := json.Marshal(daggerheartpayload.DamageAppliedPayload{
		CharacterID:    ids.CharacterID("char-1"),
		ResistPhysical: false,
	})
	eventClient := &fakeEventClient{
		listEvents: func(context.Context, *gamev1.ListEventsRequest, ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
			return &gamev1.ListEventsResponse{
				Events: []*gamev1.Event{
					{Seq: 2, PayloadJson: payloadJSON},
				},
			}, nil
		},
	}

	r := newTestRunner(scenarioEnv{eventClient: eventClient})
	state := &scenarioState{campaignID: "camp-1", ownerParticipantID: "owner-1"}
	err := r.assertDamageFlags(context.Background(), state, 0, "char-1", map[string]any{"resist_physical": true})
	if err == nil || !strings.Contains(err.Error(), "resist_physical = false, want true") {
		t.Fatalf("expected resist mismatch, got %v", err)
	}
}

func TestResolveOutcomeSeed(t *testing.T) {
	seed, err := resolveOutcomeSeed(map[string]any{}, "outcome", 12, 77)
	if err != nil {
		t.Fatalf("resolveOutcomeSeed fallback: %v", err)
	}
	if seed != 77 {
		t.Fatalf("seed = %d, want %d", seed, 77)
	}

	seed, err = resolveOutcomeSeed(map[string]any{"outcome": "fear"}, "outcome", 12, 77)
	if err != nil {
		t.Fatalf("resolveOutcomeSeed hinted: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := 12; return &d }(),
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if !matchesOutcomeHint(result, "fear") {
		t.Fatalf("seed %d produced %v, want fear", seed, result.Outcome)
	}
}

func TestFirstPlayerWithoutCharacter(t *testing.T) {
	got := firstPlayerWithoutCharacter([]string{"player-1", "player-2"}, map[string]int{"player-1": 1, "player-2": 0})
	if got != "player-2" {
		t.Fatalf("participant = %q, want %q", got, "player-2")
	}
	if got := firstPlayerWithoutCharacter([]string{"player-1"}, map[string]int{"player-1": 2}); got != "" {
		t.Fatalf("participant = %q, want empty", got)
	}
}

func TestEnsureSessionStartReadinessCreatesMissingParticipantAndCharacter(t *testing.T) {
	var createdParticipants []*gamev1.CreateParticipantRequest
	var createdCharacters []*gamev1.CreateCharacterRequest
	var setControlRequests []*gamev1.SetDefaultControlRequest
	var appliedWorkflows []*gamev1.ApplyCharacterCreationWorkflowRequest

	r := newTestRunner(scenarioEnv{
		participantClient: &fakeParticipantClient{
			listParticipants: func(context.Context, *gamev1.ListParticipantsRequest, ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
				return &gamev1.ListParticipantsResponse{}, nil
			},
			create: func(_ context.Context, req *gamev1.CreateParticipantRequest, _ ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error) {
				createdParticipants = append(createdParticipants, req)
				return &gamev1.CreateParticipantResponse{Participant: &gamev1.Participant{Id: "player-1"}}, nil
			},
		},
		characterClient: &fakeCharacterClient{
			listCharacters: func(context.Context, *gamev1.ListCharactersRequest, ...grpc.CallOption) (*gamev1.ListCharactersResponse, error) {
				return &gamev1.ListCharactersResponse{}, nil
			},
			create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
				createdCharacters = append(createdCharacters, req)
				return &gamev1.CreateCharacterResponse{Character: &gamev1.Character{Id: "char-1"}}, nil
			},
			setDefaultControl: func(_ context.Context, req *gamev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
				setControlRequests = append(setControlRequests, req)
				return &gamev1.SetDefaultControlResponse{}, nil
			},
			applyWorkflow: func(_ context.Context, req *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
				appliedWorkflows = append(appliedWorkflows, req)
				return &gamev1.ApplyCharacterCreationWorkflowResponse{}, nil
			},
			getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
				return &gamev1.GetCharacterSheetResponse{
					Profile: &gamev1.CharacterProfile{
						SystemProfile: &gamev1.CharacterProfile_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartProfile{
								ClassId:    scenarioReadinessClassID,
								SubclassId: scenarioReadinessSubclassID,
							},
						},
					},
					State: &gamev1.CharacterState{
						SystemState: &gamev1.CharacterState_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartCharacterState{
								Hp:      6,
								HopeMax: 2,
							},
						},
					},
				}, nil
			},
		},
	})
	state := &scenarioState{
		campaignID:         "camp-1",
		ownerParticipantID: "owner-1",
		campaignSystem:     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	if err := r.ensureSessionStartReadiness(context.Background(), state); err != nil {
		t.Fatalf("ensureSessionStartReadiness: %v", err)
	}
	if len(createdParticipants) != 1 {
		t.Fatalf("created participants = %d, want 1", len(createdParticipants))
	}
	if len(createdCharacters) != 1 {
		t.Fatalf("created characters = %d, want 1", len(createdCharacters))
	}
	if len(setControlRequests) != 1 || setControlRequests[0].GetParticipantId().GetValue() != "player-1" {
		t.Fatalf("set default control = %+v, want participant player-1", setControlRequests)
	}
	if len(appliedWorkflows) != 1 || appliedWorkflows[0].GetCharacterId() != "char-1" {
		t.Fatalf("applied workflows = %+v, want readiness workflow for char-1", appliedWorkflows)
	}
}

func TestEnsureSessionStartReadinessAssignsUnownedCharacterToPlayer(t *testing.T) {
	var setControlRequests []*gamev1.SetDefaultControlRequest

	r := newTestRunner(scenarioEnv{
		participantClient: &fakeParticipantClient{
			listParticipants: func(context.Context, *gamev1.ListParticipantsRequest, ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
				return &gamev1.ListParticipantsResponse{
					Participants: []*gamev1.Participant{
						{Id: "player-1", Role: gamev1.ParticipantRole_PLAYER},
					},
				}, nil
			},
		},
		characterClient: &fakeCharacterClient{
			listCharacters: func(context.Context, *gamev1.ListCharactersRequest, ...grpc.CallOption) (*gamev1.ListCharactersResponse, error) {
				return &gamev1.ListCharactersResponse{
					Characters: []*gamev1.Character{
						{Id: "char-1"},
					},
				}, nil
			},
			setDefaultControl: func(_ context.Context, req *gamev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
				setControlRequests = append(setControlRequests, req)
				return &gamev1.SetDefaultControlResponse{}, nil
			},
		},
	})
	state := &scenarioState{
		campaignID:         "camp-1",
		ownerParticipantID: "owner-1",
	}

	if err := r.ensureSessionStartReadiness(context.Background(), state); err != nil {
		t.Fatalf("ensureSessionStartReadiness: %v", err)
	}
	if len(setControlRequests) != 1 {
		t.Fatalf("set default control calls = %d, want 1", len(setControlRequests))
	}
	if setControlRequests[0].GetCharacterId() != "char-1" || setControlRequests[0].GetParticipantId().GetValue() != "player-1" {
		t.Fatalf("set control request = %+v, want char-1 -> player-1", setControlRequests[0])
	}
}

func TestEnsureSessionStartReadinessIgnoresUnimplementedListCalls(t *testing.T) {
	r := newTestRunner(scenarioEnv{
		participantClient: &fakeParticipantClient{
			listParticipants: func(context.Context, *gamev1.ListParticipantsRequest, ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
				return nil, status.Error(codes.Unimplemented, "not implemented")
			},
		},
		characterClient: &fakeCharacterClient{},
	})
	state := &scenarioState{
		campaignID:         "camp-1",
		ownerParticipantID: "owner-1",
	}

	if err := r.ensureSessionStartReadiness(context.Background(), state); err != nil {
		t.Fatalf("ensureSessionStartReadiness: %v", err)
	}
}

func TestScenarioCharacterNeedsReadiness(t *testing.T) {
	r := newTestRunner(scenarioEnv{
		characterClient: &fakeCharacterClient{
			getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
				return &gamev1.GetCharacterSheetResponse{
					Profile: &gamev1.CharacterProfile{
						SystemProfile: &gamev1.CharacterProfile_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartProfile{},
						},
					},
				}, nil
			},
		},
	})

	needsReadiness, err := r.scenarioCharacterNeedsReadiness(context.Background(), &scenarioState{}, "char-1")
	if err != nil {
		t.Fatalf("scenarioCharacterNeedsReadiness unspecified: %v", err)
	}
	if needsReadiness {
		t.Fatal("needsReadiness = true, want false when no game system is set")
	}

	needsReadiness, err = r.scenarioCharacterNeedsReadiness(context.Background(), &scenarioState{
		campaignID:     "camp-1",
		campaignSystem: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}, "char-1")
	if err != nil {
		t.Fatalf("scenarioCharacterNeedsReadiness daggerheart: %v", err)
	}
	if !needsReadiness {
		t.Fatal("needsReadiness = false, want true for missing class/subclass")
	}
}

func TestDaggerheartCharacterNeedsReadiness(t *testing.T) {
	tests := []struct {
		name    string
		client  *fakeCharacterClient
		want    bool
		wantErr string
	}{
		{
			name: "sheet lookup error",
			client: &fakeCharacterClient{
				getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
					return nil, status.Error(codes.Internal, "boom")
				},
			},
			wantErr: "get character sheet for readiness check",
		},
		{
			name: "missing profile",
			client: &fakeCharacterClient{
				getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
					return &gamev1.GetCharacterSheetResponse{}, nil
				},
			},
			want: true,
		},
		{
			name: "missing subclass",
			client: &fakeCharacterClient{
				getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
					return &gamev1.GetCharacterSheetResponse{
						Profile: &gamev1.CharacterProfile{
							SystemProfile: &gamev1.CharacterProfile_Daggerheart{
								Daggerheart: &daggerheartv1.DaggerheartProfile{ClassId: "class.guardian"},
							},
						},
					}, nil
				},
			},
			want: true,
		},
		{
			name: "ready profile",
			client: &fakeCharacterClient{
				getSheet: func(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
					return &gamev1.GetCharacterSheetResponse{
						Profile: &gamev1.CharacterProfile{
							SystemProfile: &gamev1.CharacterProfile_Daggerheart{
								Daggerheart: &daggerheartv1.DaggerheartProfile{ClassId: "class.guardian", SubclassId: "subclass.stalwart"},
							},
						},
					}, nil
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRunner(scenarioEnv{characterClient: tc.client})
			got, err := r.daggerheartCharacterNeedsReadiness(context.Background(), &scenarioState{campaignID: "camp-1"}, "char-1")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("daggerheartCharacterNeedsReadiness: %v", err)
			}
			if got != tc.want {
				t.Fatalf("needsReadiness = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestChooseActionSeed_NoHint(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{}, 12)
	if err != nil || seed != 42 {
		t.Fatalf("want 42, got %d, err=%v", seed, err)
	}
}

func TestChooseActionSeed_FearHint(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{"outcome": "fear"}, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the seed actually produces a fear outcome
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := 12; return &d }(),
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if !matchesOutcomeHint(result, "fear") {
		t.Fatalf("seed %d produced %v, not fear", seed, result.Outcome)
	}
}

func TestChooseActionSeed_TotalHint(t *testing.T) {
	difficulty := 12
	seed, err := chooseActionSeed(map[string]any{"total": 10}, difficulty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := difficulty; return &d }(),
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != 10 {
		t.Fatalf("seed %d produced total %d, want 10", seed, result.Total)
	}
}

func TestChooseActionSeed_TotalAndOutcomeHint(t *testing.T) {
	difficulty := 12
	seed, err := chooseActionSeed(map[string]any{"total": 12, "outcome": "hope"}, difficulty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := difficulty; return &d }(),
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != 12 {
		t.Fatalf("seed %d produced total %d, want 12", seed, result.Total)
	}
	if !matchesOutcomeHint(result, "hope") {
		t.Fatalf("seed %d produced %v, not hope", seed, result.Outcome)
	}
}

func TestChooseActionSeed_TotalAndAdvantage(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{"total": 30, "advantage": 1}, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := 12; return &d }(),
		Advantage:  1,
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != 30 {
		t.Fatalf("seed %d produced total %d, want 30", seed, result.Total)
	}
}

func TestChooseActionSeed_TotalAndDisadvantage(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{"total": -4, "disadvantage": 1}, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty:   func() *int { d := 12; return &d }(),
		Disadvantage: 1,
		Seed:         int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != -4 {
		t.Fatalf("seed %d produced total %d, want -4", seed, result.Total)
	}
}

func TestChooseActionSeed_TotalAndModifiers(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{
		"total": 30,
		"modifiers": []any{
			map[string]any{"source": "experience", "value": 10},
		},
	}, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := 12; return &d }(),
		Modifier:   10,
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != 30 {
		t.Fatalf("seed %d produced total %d, want 30", seed, result.Total)
	}
}

func TestChooseActionSeed_TotalAndFlatModifier(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{"total": 30, "modifier": 10}, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Difficulty: func() *int { d := 12; return &d }(),
		Modifier:   10,
		Seed:       int64(seed),
	})
	if err != nil {
		t.Fatalf("roll error: %v", err)
	}
	if result.Total != 30 {
		t.Fatalf("seed %d produced total %d, want 30", seed, result.Total)
	}
}

func TestApplyAdversaryDamageSeverityDowngradeBeforeThresholds(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	current := &daggerheartv1.DaggerheartAdversary{
		Id:              "adv-ranger",
		Hp:              6,
		Armor:           0,
		MajorThreshold:  2,
		SevereThreshold: 4,
	}
	var damageReq *daggerheartv1.DaggerheartApplyAdversaryDamageRequest
	dhClient.getAdversary = func(_ context.Context, _ *daggerheartv1.DaggerheartGetAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartGetAdversaryResponse, error) {
		snapshot := *current
		return &daggerheartv1.DaggerheartGetAdversaryResponse{Adversary: &snapshot}, nil
	}
	dhClient.applyAdversaryDamage = func(_ context.Context, req *daggerheartv1.DaggerheartApplyAdversaryDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryDamageResponse, error) {
		damageReq = req
		current.Hp = 4
		return &daggerheartv1.DaggerheartApplyAdversaryDamageResponse{AdversaryId: req.GetAdversaryId(), Adversary: current}, nil
	}

	runner := quietRunner(env)
	state := testState()
	applied, err := runner.applyAdversaryDamage(
		context.Background(),
		state,
		"adv-ranger",
		"Ranger",
		&daggerheartv1.SessionDamageRollResponse{Total: 4},
		map[string]any{
			"damage_type":        "physical",
			"severity_downgrade": 1,
		},
	)
	if err != nil {
		t.Fatalf("applyAdversaryDamage: %v", err)
	}
	if !applied {
		t.Fatal("expected damage to be applied")
	}
	if damageReq == nil || damageReq.GetDamage() == nil {
		t.Fatal("expected adversary damage request")
	}
	if got := damageReq.GetDamage().GetAmount(); got != 4 {
		t.Fatalf("damage amount = %d, want 4 with one-step severity downgrade", got)
	}
}
