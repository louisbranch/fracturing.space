package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
)

func TestResolveDeathMoveBlazeOfGlory(t *testing.T) {
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveBlazeOfGlory,
		Level:     1,
		HP:        0,
		HPMax:     6,
		Hope:      2,
		HopeMax:   HopeMax,
		Stress:    2,
		StressMax: 6,
		Seed:      1,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove returned error: %v", err)
	}
	if outcome.LifeState != LifeStateBlazeOfGlory {
		t.Fatalf("life_state = %q, want %q", outcome.LifeState, LifeStateBlazeOfGlory)
	}
}

func TestResolveDeathMoveAvoidDeath(t *testing.T) {
	seed := int64(3)
	roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 1}}, Seed: seed})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	hopeDie := roll.Rolls[0].Results[0]
	level := 2
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveAvoidDeath,
		Level:     level,
		HP:        0,
		HPMax:     6,
		Hope:      2,
		HopeMax:   HopeMax,
		Stress:    3,
		StressMax: 6,
		Seed:      seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove returned error: %v", err)
	}
	if outcome.LifeState != LifeStateUnconscious {
		t.Fatalf("life_state = %q, want %q", outcome.LifeState, LifeStateUnconscious)
	}
	if outcome.HopeDie == nil || *outcome.HopeDie != hopeDie {
		t.Fatalf("hope_die = %v, want %d", outcome.HopeDie, hopeDie)
	}
	wantScar := hopeDie <= level
	if outcome.ScarGained != wantScar {
		t.Fatalf("scar_gained = %t, want %t", outcome.ScarGained, wantScar)
	}
	if wantScar && outcome.HopeMaxAfter != HopeMax-1 {
		t.Fatalf("hope_max_after = %d, want %d", outcome.HopeMaxAfter, HopeMax-1)
	}
}

func TestResolveDeathMoveRiskItAllHopeWins(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope > fear })
	roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 2}}, Seed: seed})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	hopeDie := roll.Rolls[0].Results[0]
	fearDie := roll.Rolls[0].Results[1]
	hpClear := 2
	stressClear := 1
	if hpClear+stressClear > hopeDie {
		hpClear = hopeDie
		stressClear = 0
	}

	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:             DeathMoveRiskItAll,
		Level:            1,
		HP:               0,
		HPMax:            6,
		Hope:             2,
		HopeMax:          HopeMax,
		Stress:           3,
		StressMax:        6,
		RiskItAllHPClear: &hpClear,
		RiskItAllStClear: &stressClear,
		Seed:             seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove returned error: %v", err)
	}
	if outcome.LifeState != LifeStateAlive {
		t.Fatalf("life_state = %q, want %q", outcome.LifeState, LifeStateAlive)
	}
	if outcome.HopeDie == nil || *outcome.HopeDie != hopeDie {
		t.Fatalf("hope_die = %v, want %d", outcome.HopeDie, hopeDie)
	}
	if outcome.FearDie == nil || *outcome.FearDie != fearDie {
		t.Fatalf("fear_die = %v, want %d", outcome.FearDie, fearDie)
	}
	if outcome.HPCleared != hpClear {
		t.Fatalf("hp_cleared = %d, want %d", outcome.HPCleared, hpClear)
	}
	if outcome.StressCleared != stressClear {
		t.Fatalf("stress_cleared = %d, want %d", outcome.StressCleared, stressClear)
	}
}

func TestResolveDeathMoveRiskItAllFearWins(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return fear > hope })
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveRiskItAll,
		Level:     1,
		HP:        0,
		HPMax:     6,
		Hope:      2,
		HopeMax:   HopeMax,
		Stress:    3,
		StressMax: 6,
		Seed:      seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove returned error: %v", err)
	}
	if outcome.LifeState != LifeStateDead {
		t.Fatalf("life_state = %q, want %q", outcome.LifeState, LifeStateDead)
	}
}

func TestResolveDeathMoveRiskItAllCrit(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope == fear })
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveRiskItAll,
		Level:     1,
		HP:        0,
		HPMax:     6,
		Hope:      2,
		HopeMax:   HopeMax,
		Stress:    3,
		StressMax: 6,
		Seed:      seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove returned error: %v", err)
	}
	if outcome.LifeState != LifeStateAlive {
		t.Fatalf("life_state = %q, want %q", outcome.LifeState, LifeStateAlive)
	}
	if outcome.HPAfter != 6 {
		t.Fatalf("hp_after = %d, want 6", outcome.HPAfter)
	}
	if outcome.StressAfter != 0 {
		t.Fatalf("stress_after = %d, want 0", outcome.StressAfter)
	}
}

func TestNormalizeDeathMove(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid blaze", " Blaze_Of_Glory ", DeathMoveBlazeOfGlory, false},
		{"valid avoid", "avoid_death", DeathMoveAvoidDeath, false},
		{"valid risk", "RISK_IT_ALL", DeathMoveRiskItAll, false},
		{"empty", "  ", "", true},
		{"invalid", "unknown_move", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDeathMove(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeLifeState(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"alive", " Alive ", LifeStateAlive, false},
		{"unconscious", "UNCONSCIOUS", LifeStateUnconscious, false},
		{"blaze", "blaze_of_glory", LifeStateBlazeOfGlory, false},
		{"dead", "dead", LifeStateDead, false},
		{"empty", "", "", true},
		{"invalid", "sleeping", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLifeState(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveDeathMoveValidationErrors(t *testing.T) {
	t.Run("invalid move", func(t *testing.T) {
		_, err := ResolveDeathMove(DeathMoveInput{Move: "invalid", Level: 1, HPMax: 6, HopeMax: HopeMax, StressMax: 6})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("zero level", func(t *testing.T) {
		_, err := ResolveDeathMove(DeathMoveInput{Move: DeathMoveBlazeOfGlory, Level: 0, HPMax: 6, HopeMax: HopeMax, StressMax: 6})
		if err == nil {
			t.Fatal("expected error for zero level")
		}
	})
	t.Run("negative hp max", func(t *testing.T) {
		_, err := ResolveDeathMove(DeathMoveInput{Move: DeathMoveBlazeOfGlory, Level: 1, HPMax: -1, HopeMax: HopeMax, StressMax: 6})
		if err == nil {
			t.Fatal("expected error for negative hp_max")
		}
	})
	t.Run("negative stress max", func(t *testing.T) {
		_, err := ResolveDeathMove(DeathMoveInput{Move: DeathMoveBlazeOfGlory, Level: 1, HPMax: 6, HopeMax: HopeMax, StressMax: -1})
		if err == nil {
			t.Fatal("expected error for negative stress_max")
		}
	})
	t.Run("invalid hope max", func(t *testing.T) {
		_, err := ResolveDeathMove(DeathMoveInput{Move: DeathMoveBlazeOfGlory, Level: 1, HPMax: 6, HopeMax: HopeMax + 1, StressMax: 6})
		if err == nil {
			t.Fatal("expected error for hope_max out of range")
		}
	})
}

func TestResolveDeathMoveRiskItAllNegativeClearValues(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope > fear })
	neg := -1
	_, err := ResolveDeathMove(DeathMoveInput{
		Move: DeathMoveRiskItAll, Level: 1, HP: 0, HPMax: 6,
		Hope: 2, HopeMax: HopeMax, Stress: 3, StressMax: 6,
		RiskItAllHPClear: &neg, Seed: seed,
	})
	if err == nil {
		t.Fatal("expected error for negative clear values")
	}
}

func TestResolveDeathMoveRiskItAllClearExceedsHopeDie(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope > fear })
	roll, _ := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 2}}, Seed: seed})
	hopeDie := roll.Rolls[0].Results[0]
	hpClear := hopeDie + 1
	_, err := ResolveDeathMove(DeathMoveInput{
		Move: DeathMoveRiskItAll, Level: 1, HP: 0, HPMax: 6,
		Hope: 2, HopeMax: HopeMax, Stress: 3, StressMax: 6,
		RiskItAllHPClear: &hpClear, Seed: seed,
	})
	if err == nil {
		t.Fatal("expected error for clear values exceeding hope die")
	}
}

func TestResolveDeathMoveRiskItAllDefaultClear(t *testing.T) {
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope > fear })
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move: DeathMoveRiskItAll, Level: 1, HP: 0, HPMax: 6,
		Hope: 2, HopeMax: HopeMax, Stress: 3, StressMax: 6, Seed: seed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.HPCleared != *outcome.HopeDie {
		t.Fatalf("expected hp_cleared = hope_die (%d), got %d", *outcome.HopeDie, outcome.HPCleared)
	}
}

func TestResolveDeathMoveAvoidDeathHopeClampedByScar(t *testing.T) {
	// Find a seed where hopeDie <= level, causing a scar (hopeMax decreases).
	// With hope at the current hopeMax, the clamp path triggers.
	level := 12 // Guarantees hopeDie <= level for any d12 roll.
	seed := int64(1)
	roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 1}}, Seed: seed})
	if err != nil {
		t.Fatalf("RollDice: %v", err)
	}
	hopeDie := roll.Rolls[0].Results[0]
	if hopeDie > level {
		t.Fatal("expected hopeDie <= level with level=12")
	}

	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveAvoidDeath,
		Level:     level,
		HP:        0,
		HPMax:     6,
		Hope:      HopeMax, // At max, scar will force clamp.
		HopeMax:   HopeMax,
		Stress:    0,
		StressMax: 6,
		Seed:      seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove: %v", err)
	}
	if !outcome.ScarGained {
		t.Fatal("expected scar gained")
	}
	if outcome.HopeMaxAfter != HopeMax-1 {
		t.Fatalf("HopeMaxAfter = %d, want %d", outcome.HopeMaxAfter, HopeMax-1)
	}
	if outcome.HopeAfter != HopeMax-1 {
		t.Fatalf("HopeAfter = %d, want %d (clamped to new max)", outcome.HopeAfter, HopeMax-1)
	}
}

func TestResolveDeathMoveRiskItAllHPClearToExactMax(t *testing.T) {
	// Set up HP so that hp + hpClear == hpMax exactly.
	seed := findDeathSeed(t, func(hope, fear int) bool { return hope > fear })
	roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 2}}, Seed: seed})
	if err != nil {
		t.Fatalf("RollDice: %v", err)
	}
	hopeDie := roll.Rolls[0].Results[0]

	// Set HP such that hp + hopeDie == hpMax exactly (i.e. the clear fills to max).
	hpMax := 6
	hp := max(hpMax-hopeDie, 0)
	hpClear := hopeDie // All budget to HP.

	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:             DeathMoveRiskItAll,
		Level:            1,
		HP:               hp,
		HPMax:            hpMax,
		Hope:             2,
		HopeMax:          HopeMax,
		Stress:           0,
		StressMax:        6,
		RiskItAllHPClear: &hpClear,
		Seed:             seed,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove: %v", err)
	}
	if outcome.LifeState != LifeStateAlive {
		t.Fatalf("life_state = %q, want alive", outcome.LifeState)
	}
	expectedHP := min(hp+hpClear, hpMax)
	if outcome.HPAfter != expectedHP {
		t.Fatalf("HPAfter = %d, want %d", outcome.HPAfter, expectedHP)
	}
}

func findDeathSeed(t *testing.T, predicate func(hope, fear int) bool) int64 {
	t.Helper()
	for seed := int64(1); seed < 5000; seed++ {
		roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 2}}, Seed: seed})
		if err != nil {
			t.Fatalf("RollDice returned error: %v", err)
		}
		hopeDie := roll.Rolls[0].Results[0]
		fearDie := roll.Rolls[0].Results[1]
		if predicate(hopeDie, fearDie) {
			return seed
		}
	}
	t.Fatal("failed to find seed")
	return 0
}
