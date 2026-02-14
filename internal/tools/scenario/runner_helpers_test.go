package scenario

import (
	"context"
	"fmt"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
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
	if err != nil || ck != daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS {
		t.Fatal("expected PROGRESS")
	}
	ck, err = parseCountdownKind("consequence")
	if err != nil || ck != daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE {
		t.Fatal("expected CONSEQUENCE")
	}
	_, err = parseCountdownKind("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCountdownDirection(t *testing.T) {
	cd, err := parseCountdownDirection("increase")
	if err != nil || cd != daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE {
		t.Fatal("expected INCREASE")
	}
	cd, err = parseCountdownDirection("decrease")
	if err != nil || cd != daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE {
		t.Fatal("expected DECREASE")
	}
	_, err = parseCountdownDirection("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDowntimeMove(t *testing.T) {
	tests := []struct {
		input string
		want  daggerheartv1.DaggerheartDowntimeMove
	}{
		{"clear_all_stress", daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS},
		{"repair_all_armor", daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR},
		{"prepare", daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE},
		{"work_on_project", daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT},
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
	char := &fakeCharacterClient{
		patchProfile: func(_ context.Context, in *gamev1.PatchCharacterProfileRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			patchedProfile = in.GetDaggerheart()
			return &gamev1.PatchCharacterProfileResponse{}, nil
		},
	}
	r := newTestRunner(scenarioEnv{characterClient: char})
	state := &scenarioState{campaignID: "c-1"}

	t.Run("defaults", func(t *testing.T) {
		err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile.Level != 1 {
			t.Fatalf("expected level=1, got %d", patchedProfile.Level)
		}
		if patchedProfile.HpMax != 6 {
			t.Fatalf("expected hp_max=6, got %d", patchedProfile.HpMax)
		}
	})
	t.Run("custom_overrides", func(t *testing.T) {
		args := map[string]any{"level": 5, "hp_max": 20, "armor": 3, "agility": 2}
		err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
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
		if patchedProfile.Agility.GetValue() != 2 {
			t.Fatalf("expected agility=2, got %d", patchedProfile.Agility.GetValue())
		}
	})
	t.Run("armor_max_overrides_armor", func(t *testing.T) {
		args := map[string]any{"armor": 3, "armor_max": 5}
		err := r.applyDefaultDaggerheartProfile(context.Background(), state, "char-1", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patchedProfile.ArmorMax.GetValue() != 5 {
			t.Fatalf("expected armor_max=5, got %d", patchedProfile.ArmorMax.GetValue())
		}
	})
}

func TestApplyOptionalCharacterState_NoOp(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "c-1"}

	err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{})
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

	err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"hp": 5})
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

func TestApplyOptionalCharacterState_InvalidLifeState(t *testing.T) {
	r := newTestRunner(scenarioEnv{})
	state := &scenarioState{campaignID: "c-1"}

	err := r.applyOptionalCharacterState(context.Background(), state, "char-1", map[string]any{"life_state": "bad"})
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
