package game

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestDaggerheartConditionsFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result, err := charactertransport.DaggerheartConditionsFromProto(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty result, got %v", result)
		}
	})

	t.Run("unspecified", func(t *testing.T) {
		_, err := charactertransport.DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
			daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED,
		})
		if err == nil {
			t.Fatal("expected error for unspecified condition")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := charactertransport.DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
			daggerheartv1.DaggerheartCondition(99),
		})
		if err == nil {
			t.Fatal("expected error for invalid condition")
		}
	})

	t.Run("valid", func(t *testing.T) {
		result, err := charactertransport.DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
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
		if result[0] != rules.ConditionHidden || result[1] != rules.ConditionRestrained || result[2] != rules.ConditionVulnerable {
			t.Fatalf("unexpected condition order: %v", result)
		}
	})
}

func TestDaggerheartConditionsToProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := charactertransport.DaggerheartConditionsToProto(nil)
		if result != nil {
			t.Fatalf("expected nil result, got %v", result)
		}
	})

	t.Run("valid", func(t *testing.T) {
		result := charactertransport.DaggerheartConditionsToProto([]string{
			rules.ConditionHidden,
			"unknown",
			rules.ConditionVulnerable,
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
		_, err := charactertransport.DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED)
		if err == nil {
			t.Fatal("expected error for unspecified life state")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := charactertransport.DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState(99))
		if err == nil {
			t.Fatal("expected error for invalid life state")
		}
	})

	t.Run("valid", func(t *testing.T) {
		cases := map[daggerheartv1.DaggerheartLifeState]string{
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:          daggerheartstate.LifeStateAlive,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:    mechanics.LifeStateUnconscious,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY: mechanics.LifeStateBlazeOfGlory,
			daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:           mechanics.LifeStateDead,
		}
		for input, expected := range cases {
			result, err := charactertransport.DaggerheartLifeStateFromProto(input)
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
			daggerheartstate.LifeStateAlive: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
			mechanics.LifeStateUnconscious:  daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
			mechanics.LifeStateBlazeOfGlory: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY,
			mechanics.LifeStateDead:         daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD,
		}
		for input, expected := range cases {
			result := charactertransport.DaggerheartLifeStateToProto(input)
			if result != expected {
				t.Fatalf("life state = %v, want %v", result, expected)
			}
		}
	})

	t.Run("unknown", func(t *testing.T) {
		result := charactertransport.DaggerheartLifeStateToProto("mystery")
		if result != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
			t.Fatalf("expected unspecified, got %v", result)
		}
	})
}
