package domain

import "testing"

func TestAckOutcomeString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		outcome AckOutcome
		want    string
	}{
		{name: "unknown", outcome: AckOutcomeUnknown, want: "unknown"},
		{name: "succeeded", outcome: AckOutcomeSucceeded, want: "succeeded"},
		{name: "retry", outcome: AckOutcomeRetry, want: "retry"},
		{name: "dead", outcome: AckOutcomeDead, want: "dead"},
		{name: "invalid", outcome: AckOutcome(99), want: "unknown"},
	}

	for _, tc := range cases {
		if got := tc.outcome.String(); got != tc.want {
			t.Fatalf("%s: String() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
