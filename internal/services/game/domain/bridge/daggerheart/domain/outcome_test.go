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
