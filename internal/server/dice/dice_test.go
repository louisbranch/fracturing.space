package dice

import (
	"errors"
	"math/rand"
	"testing"
)

// TestRollDiceReturnsResults ensures roll results are deterministic and aggregated.
func TestRollDiceReturnsResults(t *testing.T) {
	result, err := RollDice(RollRequest{
		Dice: []DiceSpec{{Sides: 12, Count: 2}},
		Seed: 0,
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	if len(result.Rolls) != 1 {
		t.Fatalf("expected 1 roll, got %d", len(result.Rolls))
	}
	if result.Total != 14 {
		t.Fatalf("expected total 14, got %d", result.Total)
	}
	if result.Rolls[0].Sides != 12 {
		t.Fatalf("expected 12-sided die, got %d", result.Rolls[0].Sides)
	}
	if len(result.Rolls[0].Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Rolls[0].Results))
	}
	if result.Rolls[0].Results[0] != 7 || result.Rolls[0].Results[1] != 7 {
		t.Fatalf("unexpected results: %v", result.Rolls[0].Results)
	}
	if result.Rolls[0].Total != 14 {
		t.Fatalf("expected roll total 14, got %d", result.Rolls[0].Total)
	}
}

// TestRollDiceHandlesMultipleSpecs ensures multiple dice specs are rolled in order.
func TestRollDiceHandlesMultipleSpecs(t *testing.T) {
	seed := int64(1)
	rng := rand.New(rand.NewSource(seed))
	first := []int{rng.Intn(6) + 1, rng.Intn(6) + 1}
	second := []int{rng.Intn(8) + 1}
	firstTotal := first[0] + first[1]
	secondTotal := second[0]

	result, err := RollDice(RollRequest{
		Dice: []DiceSpec{
			{Sides: 6, Count: 2},
			{Sides: 8, Count: 1},
		},
		Seed: seed,
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	if len(result.Rolls) != 2 {
		t.Fatalf("expected 2 rolls, got %d", len(result.Rolls))
	}
	if result.Rolls[0].Total != firstTotal || result.Rolls[1].Total != secondTotal {
		t.Fatalf("unexpected roll totals: %+v", result.Rolls)
	}
	if result.Total != firstTotal+secondTotal {
		t.Fatalf("expected total %d, got %d", firstTotal+secondTotal, result.Total)
	}
}

// TestRollDiceRejectsMissingDice ensures empty requests return an error.
func TestRollDiceRejectsMissingDice(t *testing.T) {
	_, err := RollDice(RollRequest{Seed: 1})
	if !errors.Is(err, ErrMissingDice) {
		t.Fatalf("RollDice error = %v, want %v", err, ErrMissingDice)
	}
}

// TestRollDiceRejectsInvalidDiceSpec ensures invalid dice specs are rejected.
func TestRollDiceRejectsInvalidDiceSpec(t *testing.T) {
	tcs := []DiceSpec{
		{Sides: 0, Count: 2},
		{Sides: -1, Count: 2},
		{Sides: 6, Count: 0},
		{Sides: 6, Count: -1},
	}

	for _, tc := range tcs {
		_, err := RollDice(RollRequest{
			Dice: []DiceSpec{tc},
			Seed: 2,
		})
		if !errors.Is(err, ErrInvalidDiceSpec) {
			t.Fatalf("RollDice(%+v) error = %v, want %v", tc, err, ErrInvalidDiceSpec)
		}
	}
}

func TestRollAction(t *testing.T) {
	diff := func(d int) *int {
		return &d
	}

	tcs := []struct {
		wantOutcome Outcome
		seed        int64
		modifier    int
		difficulty  *int
		wantHope    int
		wantFear    int
		wantTotal   int
	}{
		{
			wantOutcome: OutcomeCriticalSuccess,
			seed:        0,
			modifier:    0,
			difficulty:  nil,
			wantHope:    7,
			wantFear:    7,
			wantTotal:   14,
		},
		{
			wantOutcome: OutcomeRollWithHope,
			seed:        1,
			modifier:    0,
			difficulty:  nil,
			wantHope:    6,
			wantFear:    4,
			wantTotal:   10,
		},
		{
			wantOutcome: OutcomeRollWithFear,
			seed:        3,
			modifier:    0,
			difficulty:  nil,
			wantHope:    5,
			wantFear:    6,
			wantTotal:   11,
		},
		{
			wantOutcome: OutcomeSuccessWithHope,
			seed:        1,
			modifier:    0,
			difficulty:  diff(9),
			wantHope:    6,
			wantFear:    4,
			wantTotal:   10,
		},
		{
			wantOutcome: OutcomeSuccessWithFear,
			seed:        3,
			modifier:    0,
			difficulty:  diff(10),
			wantHope:    5,
			wantFear:    6,
			wantTotal:   11,
		},
		{
			wantOutcome: OutcomeFailureWithHope,
			seed:        1,
			modifier:    -1,
			difficulty:  diff(10),
			wantHope:    6,
			wantFear:    4,
			wantTotal:   9,
		},
		{
			wantOutcome: OutcomeFailureWithFear,
			seed:        3,
			modifier:    -2,
			difficulty:  diff(10),
			wantHope:    5,
			wantFear:    6,
			wantTotal:   9,
		},
	}

	for _, tc := range tcs {
		result, err := RollAction(ActionRequest{
			Modifier:   tc.modifier,
			Difficulty: tc.difficulty,
			Seed:       tc.seed,
		})
		if err != nil {
			t.Fatalf("RollAction returned error: %v", err)
		}
		if result.Hope != tc.wantHope || result.Fear != tc.wantFear || result.Total != tc.wantTotal || result.Outcome != tc.wantOutcome {
			t.Errorf("RollAction(%d, %v) = (%d, %d, %d, %v), want (%d, %d, %d, %v)", tc.modifier, tc.difficulty, result.Hope, result.Fear, result.Total, result.Outcome, tc.wantHope, tc.wantFear, tc.wantTotal, tc.wantOutcome)
		}
	}
}

func TestRollActionRejectsNegativeDifficulty(t *testing.T) {
	difficulty := -1
	_, err := RollAction(ActionRequest{
		Modifier:   0,
		Difficulty: &difficulty,
		Seed:       0,
	})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("RollAction error = %v, want %v", err, ErrInvalidDifficulty)
	}
}
