//go:build scenario

package game

import "testing"

func TestMatchesOutcomeExpectation(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		match    bool
	}{
		{
			name:     "hope matches success with hope",
			expected: "hope",
			actual:   "OUTCOME_SUCCESS_WITH_HOPE",
			match:    true,
		},
		{
			name:     "fear matches roll with fear",
			expected: "fear",
			actual:   "OUTCOME_ROLL_WITH_FEAR",
			match:    true,
		},
		{
			name:     "critical matches critical success",
			expected: "critical",
			actual:   "OUTCOME_CRITICAL_SUCCESS",
			match:    true,
		},
		{
			name:     "explicit outcome code matches",
			expected: "OUTCOME_FAILURE_WITH_HOPE",
			actual:   "OUTCOME_FAILURE_WITH_HOPE",
			match:    true,
		},
		{
			name:     "shorthand outcome code matches",
			expected: "success_with_fear",
			actual:   "OUTCOME_SUCCESS_WITH_FEAR",
			match:    true,
		},
		{
			name:     "unknown expectation fails",
			expected: "mystery",
			actual:   "OUTCOME_SUCCESS_WITH_HOPE",
			match:    false,
		},
	}

	for _, tt := range tests {
		result, _ := matchesOutcomeExpectation(tt.expected, tt.actual)
		if result != tt.match {
			t.Fatalf("%s: match = %v, want %v", tt.name, result, tt.match)
		}
	}
}
