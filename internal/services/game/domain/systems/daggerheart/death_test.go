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
