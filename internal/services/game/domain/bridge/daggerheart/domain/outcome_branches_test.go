package domain

import (
	"errors"
	"testing"
)

func TestEvaluateOutcome_BranchMatrix(t *testing.T) {
	difficulty := 14
	tests := []struct {
		name        string
		request     OutcomeRequest
		wantOutcome Outcome
		wantMeets   bool
		wantCrit    bool
	}{
		{
			name: "critical success overrides difficulty",
			request: OutcomeRequest{
				Hope:       6,
				Fear:       6,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeCriticalSuccess,
			wantMeets:   true,
			wantCrit:    true,
		},
		{
			name: "roll with hope when no difficulty",
			request: OutcomeRequest{
				Hope: 9,
				Fear: 2,
			},
			wantOutcome: OutcomeRollWithHope,
			wantMeets:   false,
		},
		{
			name: "roll with fear when no difficulty",
			request: OutcomeRequest{
				Hope: 2,
				Fear: 9,
			},
			wantOutcome: OutcomeRollWithFear,
			wantMeets:   false,
		},
		{
			name: "success with hope when difficulty met",
			request: OutcomeRequest{
				Hope:       8,
				Fear:       4,
				Modifier:   2,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeSuccessWithHope,
			wantMeets:   true,
		},
		{
			name: "success with fear when difficulty met",
			request: OutcomeRequest{
				Hope:       5,
				Fear:       8,
				Modifier:   1,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeSuccessWithFear,
			wantMeets:   true,
		},
		{
			name: "failure with hope when difficulty not met",
			request: OutcomeRequest{
				Hope:       8,
				Fear:       4,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeFailureWithHope,
			wantMeets:   false,
		},
		{
			name: "failure with fear when difficulty not met",
			request: OutcomeRequest{
				Hope:       4,
				Fear:       8,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeFailureWithFear,
			wantMeets:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateOutcome(tc.request)
			if err != nil {
				t.Fatalf("EvaluateOutcome returned error: %v", err)
			}
			if result.Outcome != tc.wantOutcome {
				t.Fatalf("outcome = %v, want %v", result.Outcome, tc.wantOutcome)
			}
			if result.MeetsDifficulty != tc.wantMeets {
				t.Fatalf("meets difficulty = %v, want %v", result.MeetsDifficulty, tc.wantMeets)
			}
			if result.IsCrit != tc.wantCrit {
				t.Fatalf("is crit = %v, want %v", result.IsCrit, tc.wantCrit)
			}
		})
	}
}

func TestEvaluateOutcome_InvalidInputs(t *testing.T) {
	if _, err := EvaluateOutcome(OutcomeRequest{Hope: 0, Fear: 5}); !errors.Is(err, ErrInvalidDualityDie) {
		t.Fatalf("invalid die error = %v, want ErrInvalidDualityDie", err)
	}
	negativeDifficulty := -1
	if _, err := EvaluateOutcome(OutcomeRequest{Hope: 6, Fear: 5, Difficulty: &negativeDifficulty}); !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("invalid difficulty error = %v, want ErrInvalidDifficulty", err)
	}
}
