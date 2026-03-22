package gametools

import "testing"

func TestNormalizeCountdownCreateInputPrefersFixedStartOverInvalidRandomizedStart(t *testing.T) {
	input := normalizeCountdownCreateInput(countdownCreateInput{
		FixedStartingValue: 1,
		RandomizedStart:    &rangeInput{Min: 0, Max: 0},
	})
	if input.RandomizedStart != nil {
		t.Fatalf("randomized_start = %#v, want nil", input.RandomizedStart)
	}
	if input.FixedStartingValue != 1 {
		t.Fatalf("fixed_starting_value = %d, want 1", input.FixedStartingValue)
	}
}
