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
		wantOutcome         Outcome
		seed                int64
		modifier            int
		difficulty          *int
		wantHope            int
		wantFear            int
		wantTotal           int
		wantIsCrit          bool
		wantMeetsDifficulty bool
	}{
		{
			wantOutcome:         OutcomeCriticalSuccess,
			seed:                0,
			modifier:            0,
			difficulty:          nil,
			wantHope:            7,
			wantFear:            7,
			wantTotal:           14,
			wantIsCrit:          true,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeRollWithHope,
			seed:                1,
			modifier:            0,
			difficulty:          nil,
			wantHope:            6,
			wantFear:            4,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeRollWithFear,
			seed:                3,
			modifier:            0,
			difficulty:          nil,
			wantHope:            5,
			wantFear:            6,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeSuccessWithHope,
			seed:                1,
			modifier:            0,
			difficulty:          diff(9),
			wantHope:            6,
			wantFear:            4,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			wantOutcome:         OutcomeSuccessWithFear,
			seed:                3,
			modifier:            0,
			difficulty:          diff(10),
			wantHope:            5,
			wantFear:            6,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			wantOutcome:         OutcomeFailureWithHope,
			seed:                1,
			modifier:            -1,
			difficulty:          diff(10),
			wantHope:            6,
			wantFear:            4,
			wantTotal:           9,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeFailureWithFear,
			seed:                3,
			modifier:            -2,
			difficulty:          diff(10),
			wantHope:            5,
			wantFear:            6,
			wantTotal:           9,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
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
		if result.Hope != tc.wantHope || result.Fear != tc.wantFear || result.Total != tc.wantTotal || result.Outcome != tc.wantOutcome || result.IsCrit != tc.wantIsCrit || result.MeetsDifficulty != tc.wantMeetsDifficulty {
			t.Errorf("RollAction(%d, %v) = (%d, %d, %d, %v, %t, %t), want (%d, %d, %d, %v, %t, %t)", tc.modifier, tc.difficulty, result.Hope, result.Fear, result.Total, result.Outcome, result.IsCrit, result.MeetsDifficulty, tc.wantHope, tc.wantFear, tc.wantTotal, tc.wantOutcome, tc.wantIsCrit, tc.wantMeetsDifficulty)
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

func TestEvaluateOutcome(t *testing.T) {
	tcs := []struct {
		name                string
		request             OutcomeRequest
		wantOutcome         Outcome
		wantTotal           int
		wantIsCrit          bool
		wantMeetsDifficulty bool
	}{
		{
			name: "critical overrides difficulty",
			request: OutcomeRequest{
				Hope:       7,
				Fear:       7,
				Modifier:   0,
				Difficulty: intPtr(12),
			},
			wantOutcome:         OutcomeCriticalSuccess,
			wantTotal:           14,
			wantIsCrit:          true,
			wantMeetsDifficulty: true,
		},
		{
			name: "success with hope",
			request: OutcomeRequest{
				Hope:       10,
				Fear:       4,
				Modifier:   1,
				Difficulty: intPtr(10),
			},
			wantOutcome:         OutcomeSuccessWithHope,
			wantTotal:           15,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			name: "failure with fear",
			request: OutcomeRequest{
				Hope:       2,
				Fear:       8,
				Modifier:   0,
				Difficulty: intPtr(11),
			},
			wantOutcome:         OutcomeFailureWithFear,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			name: "roll with hope",
			request: OutcomeRequest{
				Hope:     9,
				Fear:     2,
				Modifier: 0,
			},
			wantOutcome:         OutcomeRollWithHope,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			name: "roll with fear",
			request: OutcomeRequest{
				Hope:     3,
				Fear:     11,
				Modifier: 0,
			},
			wantOutcome:         OutcomeRollWithFear,
			wantTotal:           14,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateOutcome(tc.request)
			if err != nil {
				t.Fatalf("EvaluateOutcome returned error: %v", err)
			}
			if result.Total != tc.wantTotal || result.Outcome != tc.wantOutcome || result.IsCrit != tc.wantIsCrit || result.MeetsDifficulty != tc.wantMeetsDifficulty {
				t.Fatalf("EvaluateOutcome = (%d, %v, %t, %t), want (%d, %v, %t, %t)", result.Total, result.Outcome, result.IsCrit, result.MeetsDifficulty, tc.wantTotal, tc.wantOutcome, tc.wantIsCrit, tc.wantMeetsDifficulty)
			}
		})
	}
}

func TestEvaluateOutcomeRejectsInvalidDice(t *testing.T) {
	_, err := EvaluateOutcome(OutcomeRequest{Hope: 0, Fear: 12})
	if !errors.Is(err, ErrInvalidDualityDie) {
		t.Fatalf("EvaluateOutcome error = %v, want %v", err, ErrInvalidDualityDie)
	}
}

func TestEvaluateOutcomeRejectsNegativeDifficulty(t *testing.T) {
	_, err := EvaluateOutcome(OutcomeRequest{Hope: 6, Fear: 5, Difficulty: intPtr(-1)})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("EvaluateOutcome error = %v, want %v", err, ErrInvalidDifficulty)
	}
}

func TestDualityProbabilityCounts(t *testing.T) {
	result, err := DualityProbability(ProbabilityRequest{Modifier: 0, Difficulty: 10})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	if result.TotalOutcomes != 144 {
		t.Fatalf("total outcomes = %d, want 144", result.TotalOutcomes)
	}
	if result.CritCount != 12 {
		t.Fatalf("crit count = %d, want 12", result.CritCount)
	}
	if result.SuccessCount+result.FailureCount != result.TotalOutcomes {
		t.Fatalf("success+failure = %d, want %d", result.SuccessCount+result.FailureCount, result.TotalOutcomes)
	}
	countSum := 0
	for _, count := range result.OutcomeCounts {
		countSum += count.Count
	}
	if countSum != result.TotalOutcomes {
		t.Fatalf("outcome count sum = %d, want %d", countSum, result.TotalOutcomes)
	}
}

func TestDualityProbabilityCritsConstant(t *testing.T) {
	first, err := DualityProbability(ProbabilityRequest{Modifier: -2, Difficulty: 8})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	second, err := DualityProbability(ProbabilityRequest{Modifier: 5, Difficulty: 18})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	if first.CritCount != second.CritCount {
		t.Fatalf("crit count changed: %d vs %d", first.CritCount, second.CritCount)
	}
}

func TestDualityProbabilityRejectsNegativeDifficulty(t *testing.T) {
	_, err := DualityProbability(ProbabilityRequest{Modifier: 0, Difficulty: -1})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("DualityProbability error = %v, want %v", err, ErrInvalidDifficulty)
	}
}

func intPtr(value int) *int {
	return &value
}
