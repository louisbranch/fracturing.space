package domain

import "testing"

func TestEvaluateOutcome_TieDifficultyCountsAsSuccess(t *testing.T) {
	difficulty := 12

	tests := []struct {
		name        string
		request     OutcomeRequest
		wantOutcome Outcome
	}{
		{
			name: "hope branch",
			request: OutcomeRequest{
				Hope:       6,
				Fear:       5,
				Modifier:   1,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeSuccessWithHope,
		},
		{
			name: "fear branch",
			request: OutcomeRequest{
				Hope:       4,
				Fear:       7,
				Modifier:   1,
				Difficulty: &difficulty,
			},
			wantOutcome: OutcomeSuccessWithFear,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateOutcome(tc.request)
			if err != nil {
				t.Fatalf("EvaluateOutcome returned error: %v", err)
			}
			if !result.MeetsDifficulty {
				t.Fatal("expected tie total to meet difficulty")
			}
			if result.Outcome != tc.wantOutcome {
				t.Fatalf("outcome = %v, want %v", result.Outcome, tc.wantOutcome)
			}
		})
	}
}

func TestOutcomeString_AllKnownOutcomes(t *testing.T) {
	tests := []struct {
		outcome Outcome
		want    string
	}{
		{outcome: OutcomeUnspecified, want: "Unspecified"},
		{outcome: OutcomeRollWithHope, want: "Roll with hope"},
		{outcome: OutcomeRollWithFear, want: "Roll with fear"},
		{outcome: OutcomeSuccessWithHope, want: "Success with hope"},
		{outcome: OutcomeSuccessWithFear, want: "Success with fear"},
		{outcome: OutcomeFailureWithHope, want: "Failure with hope"},
		{outcome: OutcomeFailureWithFear, want: "Failure with fear"},
		{outcome: OutcomeCriticalSuccess, want: "Critical success"},
	}
	for _, tc := range tests {
		if got := tc.outcome.String(); got != tc.want {
			t.Fatalf("Outcome(%d).String() = %q, want %q", tc.outcome, got, tc.want)
		}
	}
}
