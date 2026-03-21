package sessionrolltransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

func TestNormalizeRollKind(t *testing.T) {
	tests := []struct {
		name string
		kind pb.RollKind
		want pb.RollKind
	}{
		{"unspecified defaults to action", pb.RollKind_ROLL_KIND_UNSPECIFIED, pb.RollKind_ROLL_KIND_ACTION},
		{"action stays action", pb.RollKind_ROLL_KIND_ACTION, pb.RollKind_ROLL_KIND_ACTION},
		{"reaction stays reaction", pb.RollKind_ROLL_KIND_REACTION, pb.RollKind_ROLL_KIND_REACTION},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRollKind(tc.kind); got != tc.want {
				t.Errorf("normalizeRollKind(%v) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

func TestNormalizeActionModifiers(t *testing.T) {
	t.Run("empty modifiers", func(t *testing.T) {
		total, entries := normalizeActionModifiers(nil)
		if total != 0 || entries != nil {
			t.Errorf("expected (0, nil), got (%d, %v)", total, entries)
		}
	})

	t.Run("single modifier", func(t *testing.T) {
		total, entries := normalizeActionModifiers([]*pb.ActionRollModifier{
			{Value: 3, Source: "experience"},
		})
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("nil modifier skipped", func(t *testing.T) {
		total, entries := normalizeActionModifiers([]*pb.ActionRollModifier{
			nil,
			{Value: 2},
		})
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("multiple modifiers sum", func(t *testing.T) {
		total, entries := normalizeActionModifiers([]*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
			{Value: -1, Source: "penalty"},
		})
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
	})
}

func TestNormalizeHopeSpendSource(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"experience", "experience"},
		{"  Help  ", "help"},
		{"Tag Team", "tag_team"},
		{"hope-feature", "hope_feature"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := normalizeHopeSpendSource(tc.input); got != tc.want {
				t.Errorf("normalizeHopeSpendSource(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHopeSpendsFromModifiers(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := hopeSpendsFromModifiers(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("experience source", func(t *testing.T) {
		spends := hopeSpendsFromModifiers([]*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
		})
		if len(spends) != 1 {
			t.Fatalf("expected 1 spend, got %d", len(spends))
		}
		if spends[0].Amount != 1 {
			t.Errorf("amount = %d, want 1", spends[0].Amount)
		}
	})

	t.Run("nil modifier skipped", func(t *testing.T) {
		spends := hopeSpendsFromModifiers([]*pb.ActionRollModifier{nil})
		if len(spends) != 0 {
			t.Errorf("expected 0 spends, got %d", len(spends))
		}
	})
}

func TestResolveRollActionKind(t *testing.T) {
	difficulty := 10
	result, genHopeFear, triggersGM, critNegates, err := resolveRoll(
		pb.RollKind_ROLL_KIND_ACTION,
		daggerheartdomain.ActionRequest{Seed: 42, Difficulty: &difficulty},
	)
	if err != nil {
		t.Fatalf("resolveRoll(ACTION) error: %v", err)
	}
	if !genHopeFear {
		t.Error("action roll should generate hope/fear")
	}
	if !triggersGM {
		t.Error("action roll should trigger GM move")
	}
	if critNegates {
		t.Error("action roll should not have crit negates")
	}
	if result.Hope == 0 && result.Fear == 0 {
		t.Error("expected non-zero dice values")
	}
}

func TestResolveRollReactionKind(t *testing.T) {
	difficulty := 10
	_, _, _, _, err := resolveRoll(
		pb.RollKind_ROLL_KIND_REACTION,
		daggerheartdomain.ActionRequest{Seed: 42, Difficulty: &difficulty},
	)
	if err != nil {
		t.Fatalf("resolveRoll(REACTION) error: %v", err)
	}
}

func TestResolveRollUnspecifiedDefaultsToAction(t *testing.T) {
	difficulty := 10
	_, genHopeFear, triggersGM, critNegates, err := resolveRoll(
		pb.RollKind_ROLL_KIND_UNSPECIFIED,
		daggerheartdomain.ActionRequest{Seed: 42, Difficulty: &difficulty},
	)
	if err != nil {
		t.Fatalf("resolveRoll(UNSPECIFIED) error: %v", err)
	}
	if !genHopeFear || !triggersGM {
		t.Error("unspecified should default to action kind")
	}
	if critNegates {
		t.Error("action roll should not have crit negates")
	}
}
