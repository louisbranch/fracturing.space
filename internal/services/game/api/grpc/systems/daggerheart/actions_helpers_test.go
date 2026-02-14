package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/metadata"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		target string
		want   bool
	}{
		{"empty target", []string{"a", "b"}, "", false},
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b"}, "z", false},
		{"empty slice", nil, "a", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsString(tc.values, tc.target); got != tc.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tc.values, tc.target, got, tc.want)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		minValue int
		maxValue int
		want     int
	}{
		{"within range", 5, 0, 10, 5},
		{"below min", -1, 0, 10, 0},
		{"above max", 15, 0, 10, 10},
		{"at min", 0, 0, 10, 0},
		{"at max", 10, 0, 10, 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := clamp(tc.value, tc.minValue, tc.maxValue); got != tc.want {
				t.Errorf("clamp(%d, %d, %d) = %d, want %d", tc.value, tc.minValue, tc.maxValue, got, tc.want)
			}
		})
	}
}

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

func TestOutcomeFlavorFromCode(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{pb.Outcome_ROLL_WITH_HOPE.String(), "HOPE"},
		{pb.Outcome_SUCCESS_WITH_HOPE.String(), "HOPE"},
		{pb.Outcome_FAILURE_WITH_HOPE.String(), "HOPE"},
		{pb.Outcome_CRITICAL_SUCCESS.String(), "HOPE"},
		{pb.Outcome_ROLL_WITH_FEAR.String(), "FEAR"},
		{pb.Outcome_SUCCESS_WITH_FEAR.String(), "FEAR"},
		{pb.Outcome_FAILURE_WITH_FEAR.String(), "FEAR"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			if got := outcomeFlavorFromCode(tc.code); got != tc.want {
				t.Errorf("outcomeFlavorFromCode(%q) = %q, want %q", tc.code, got, tc.want)
			}
		})
	}
}

func TestOutcomeSuccessFromCode(t *testing.T) {
	tests := []struct {
		code    string
		success bool
		known   bool
	}{
		{pb.Outcome_SUCCESS_WITH_HOPE.String(), true, true},
		{pb.Outcome_SUCCESS_WITH_FEAR.String(), true, true},
		{pb.Outcome_CRITICAL_SUCCESS.String(), true, true},
		{pb.Outcome_FAILURE_WITH_HOPE.String(), false, true},
		{pb.Outcome_FAILURE_WITH_FEAR.String(), false, true},
		{pb.Outcome_ROLL_WITH_HOPE.String(), false, true},
		{pb.Outcome_ROLL_WITH_FEAR.String(), false, true},
		{"unknown", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			success, known := outcomeSuccessFromCode(tc.code)
			if success != tc.success || known != tc.known {
				t.Errorf("outcomeSuccessFromCode(%q) = (%v, %v), want (%v, %v)",
					tc.code, success, known, tc.success, tc.known)
			}
		})
	}
}

func TestOutcomeCodeToProto(t *testing.T) {
	outcomes := []pb.Outcome{
		pb.Outcome_ROLL_WITH_HOPE,
		pb.Outcome_ROLL_WITH_FEAR,
		pb.Outcome_SUCCESS_WITH_HOPE,
		pb.Outcome_SUCCESS_WITH_FEAR,
		pb.Outcome_FAILURE_WITH_HOPE,
		pb.Outcome_FAILURE_WITH_FEAR,
		pb.Outcome_CRITICAL_SUCCESS,
	}
	for _, outcome := range outcomes {
		t.Run(outcome.String(), func(t *testing.T) {
			if got := outcomeCodeToProto(outcome.String()); got != outcome {
				t.Errorf("outcomeCodeToProto(%q) = %v, want %v", outcome.String(), got, outcome)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		if got := outcomeCodeToProto("invalid"); got != pb.Outcome_OUTCOME_UNSPECIFIED {
			t.Errorf("outcomeCodeToProto(invalid) = %v, want UNSPECIFIED", got)
		}
	})
}

func TestOutcomeFromSystemData(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		fallback string
		want     string
	}{
		{"nil data uses fallback", nil, "fallback", "fallback"},
		{"missing key uses fallback", map[string]any{"other": "x"}, "fallback", "fallback"},
		{"found outcome", map[string]any{"outcome": "ROLL_WITH_HOPE"}, "", "ROLL_WITH_HOPE"},
		{"wrong type uses fallback", map[string]any{"outcome": 42}, "default", "default"},
		{"trims fallback", nil, "  trimme  ", "trimme"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := outcomeFromSystemData(tc.data, tc.fallback); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRollKindFromSystemData(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want pb.RollKind
	}{
		{"nil data", nil, pb.RollKind_ROLL_KIND_ACTION},
		{"missing key", map[string]any{}, pb.RollKind_ROLL_KIND_ACTION},
		{"wrong type", map[string]any{"roll_kind": 42}, pb.RollKind_ROLL_KIND_ACTION},
		{"action", map[string]any{"roll_kind": pb.RollKind_ROLL_KIND_ACTION.String()}, pb.RollKind_ROLL_KIND_ACTION},
		{"reaction", map[string]any{"roll_kind": pb.RollKind_ROLL_KIND_REACTION.String()}, pb.RollKind_ROLL_KIND_REACTION},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rollKindFromSystemData(tc.data); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoolFromSystemData(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		key      string
		fallback bool
		want     bool
	}{
		{"nil data", nil, "key", true, true},
		{"missing key", map[string]any{}, "key", false, false},
		{"wrong type", map[string]any{"key": "nope"}, "key", false, false},
		{"true value", map[string]any{"key": true}, "key", false, true},
		{"false value", map[string]any{"key": false}, "key", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := boolFromSystemData(tc.data, tc.key, tc.fallback); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCritFromSystemData(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		outcome string
		want    bool
	}{
		{"from data", map[string]any{"crit": true}, "", true},
		{"from data false", map[string]any{"crit": false}, pb.Outcome_CRITICAL_SUCCESS.String(), false},
		{"from outcome string", nil, pb.Outcome_CRITICAL_SUCCESS.String(), true},
		{"from outcome short string", nil, "CRITICAL_SUCCESS", true},
		{"neither", nil, "ROLL_WITH_HOPE", false},
		{"wrong type in data", map[string]any{"crit": "yes"}, pb.Outcome_CRITICAL_SUCCESS.String(), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := critFromSystemData(tc.data, tc.outcome); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStringFromSystemData(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		key  string
		want string
	}{
		{"nil data", nil, "key", ""},
		{"missing key", map[string]any{}, "key", ""},
		{"wrong type", map[string]any{"key": 42}, "key", ""},
		{"found", map[string]any{"key": " hello "}, "key", "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stringFromSystemData(tc.data, tc.key); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDaggerheartSeverityToString(t *testing.T) {
	tests := []struct {
		severity daggerheart.DamageSeverity
		want     string
	}{
		{daggerheart.DamageMinor, "minor"},
		{daggerheart.DamageMajor, "major"},
		{daggerheart.DamageSevere, "severe"},
		{daggerheart.DamageMassive, "massive"},
		{daggerheart.DamageSeverity(99), "none"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := daggerheartSeverityToString(tc.severity); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDaggerheartDamageTypeToString(t *testing.T) {
	tests := []struct {
		dt   pb.DaggerheartDamageType
		want string
	}{
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL, "physical"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC, "magic"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED, "mixed"},
		{pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED, "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := daggerheartDamageTypeToString(tc.dt); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDaggerheartDowntimeMoveFromProto(t *testing.T) {
	tests := []struct {
		name    string
		move    pb.DaggerheartDowntimeMove
		want    daggerheart.DowntimeMove
		wantErr bool
	}{
		{"clear stress", pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS, daggerheart.DowntimeClearAllStress, false},
		{"repair armor", pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR, daggerheart.DowntimeRepairAllArmor, false},
		{"prepare", pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE, daggerheart.DowntimePrepare, false},
		{"work on project", pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT, daggerheart.DowntimeWorkOnProject, false},
		{"unspecified", pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED, daggerheart.DowntimePrepare, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := daggerheartDowntimeMoveFromProto(tc.move)
			if (err != nil) != tc.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDaggerheartDowntimeMoveToString(t *testing.T) {
	tests := []struct {
		move daggerheart.DowntimeMove
		want string
	}{
		{daggerheart.DowntimeClearAllStress, "clear_all_stress"},
		{daggerheart.DowntimeRepairAllArmor, "repair_all_armor"},
		{daggerheart.DowntimePrepare, "prepare"},
		{daggerheart.DowntimeWorkOnProject, "work_on_project"},
		{daggerheart.DowntimeMove(99), "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := daggerheartDowntimeMoveToString(tc.move); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDamageDiceFromProto(t *testing.T) {
	t.Run("empty specs", func(t *testing.T) {
		_, err := damageDiceFromProto(nil)
		if err != dice.ErrMissingDice {
			t.Errorf("expected ErrMissingDice, got %v", err)
		}
	})

	t.Run("nil spec", func(t *testing.T) {
		_, err := damageDiceFromProto([]*pb.DiceSpec{nil})
		if err != dice.ErrInvalidDiceSpec {
			t.Errorf("expected ErrInvalidDiceSpec, got %v", err)
		}
	})

	t.Run("invalid sides", func(t *testing.T) {
		_, err := damageDiceFromProto([]*pb.DiceSpec{{Sides: 0, Count: 1}})
		if err != dice.ErrInvalidDiceSpec {
			t.Errorf("expected ErrInvalidDiceSpec, got %v", err)
		}
	})

	t.Run("valid spec", func(t *testing.T) {
		result, err := damageDiceFromProto([]*pb.DiceSpec{{Sides: 6, Count: 2}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0].Sides != 6 || result[0].Count != 2 {
			t.Errorf("unexpected result: %v", result)
		}
	})
}

func TestDiceRollsToProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := diceRollsToProto(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("valid rolls", func(t *testing.T) {
		rolls := []dice.Roll{
			{Sides: 6, Results: []int{3, 4}, Total: 7},
		}
		protos := diceRollsToProto(rolls)
		if len(protos) != 1 {
			t.Fatalf("expected 1 roll, got %d", len(protos))
		}
		if protos[0].Sides != 6 || protos[0].Total != 7 {
			t.Errorf("roll mismatch: %v", protos[0])
		}
	})
}

func TestWithCampaignSessionMetadata(t *testing.T) {
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		t.Fatal("expected incoming metadata")
	}
	if got := md.Get(grpcmeta.CampaignIDHeader); len(got) == 0 || got[0] != "camp-1" {
		t.Errorf("campaign ID = %v, want camp-1", got)
	}
	if got := md.Get(grpcmeta.SessionIDHeader); len(got) == 0 || got[0] != "sess-1" {
		t.Errorf("session ID = %v, want sess-1", got)
	}
}
