package game

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func TestDaggerheartConditionsFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result, err := daggerheartConditionsFromProto(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty result, got %v", result)
		}
	})

	t.Run("unspecified", func(t *testing.T) {
		_, err := daggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
			daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED,
		})
		if err == nil {
			t.Fatal("expected error for unspecified condition")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := daggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
			daggerheartv1.DaggerheartCondition(99),
		})
		if err == nil {
			t.Fatal("expected error for invalid condition")
		}
	})

	t.Run("valid", func(t *testing.T) {
		result, err := daggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
			daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
			daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED,
			daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 conditions, got %d", len(result))
		}
		if result[0] != daggerheart.ConditionHidden || result[1] != daggerheart.ConditionRestrained || result[2] != daggerheart.ConditionVulnerable {
			t.Fatalf("unexpected condition order: %v", result)
		}
	})
}

func TestDaggerheartConditionsToProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := daggerheartConditionsToProto(nil)
		if result != nil {
			t.Fatalf("expected nil result, got %v", result)
		}
	})

	t.Run("valid", func(t *testing.T) {
		result := daggerheartConditionsToProto([]string{
			daggerheart.ConditionHidden,
			"unknown",
			daggerheart.ConditionVulnerable,
		})
		if len(result) != 2 {
			t.Fatalf("expected 2 conditions, got %d", len(result))
		}
		if result[0] != daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN || result[1] != daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE {
			t.Fatalf("unexpected proto conditions: %v", result)
		}
	})
}

func TestDaggerheartLifeStateFromProto(t *testing.T) {
	t.Run("unspecified", func(t *testing.T) {
		_, err := daggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED)
		if err == nil {
			t.Fatal("expected error for unspecified life state")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := daggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState(99))
		if err == nil {
			t.Fatal("expected error for invalid life state")
		}
	})

	t.Run("valid", func(t *testing.T) {
		cases := map[daggerheartv1.DaggerheartLifeState]string{
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:          daggerheart.LifeStateAlive,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:    daggerheart.LifeStateUnconscious,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY: daggerheart.LifeStateBlazeOfGlory,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:           daggerheart.LifeStateDead,
		}
		for input, expected := range cases {
			result, err := daggerheartLifeStateFromProto(input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != expected {
				t.Fatalf("life state = %q, want %q", result, expected)
			}
		}
	})
}

func TestDaggerheartLifeStateToProto(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := map[string]daggerheartv1.DaggerheartLifeState{
			daggerheart.LifeStateAlive:        daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
			daggerheart.LifeStateUnconscious:  daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
			daggerheart.LifeStateBlazeOfGlory: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY,
			daggerheart.LifeStateDead:         daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD,
		}
		for input, expected := range cases {
			result := daggerheartLifeStateToProto(input)
			if result != expected {
				t.Fatalf("life state = %v, want %v", result, expected)
			}
		}
	})

	t.Run("unknown", func(t *testing.T) {
		result := daggerheartLifeStateToProto("mystery")
		if result != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
			t.Fatalf("expected unspecified, got %v", result)
		}
	})
}
