package provider

import "testing"

func TestUsageIsZeroAndAdd(t *testing.T) {
	var empty Usage
	if !empty.IsZero() {
		t.Fatal("empty usage should be zero")
	}

	got := Usage{
		InputTokens:     3,
		OutputTokens:    5,
		ReasoningTokens: 2,
		TotalTokens:     8,
	}.Add(Usage{
		InputTokens:     7,
		OutputTokens:    11,
		ReasoningTokens: 4,
		TotalTokens:     18,
	})

	want := Usage{
		InputTokens:     10,
		OutputTokens:    16,
		ReasoningTokens: 6,
		TotalTokens:     26,
	}
	if got != want {
		t.Fatalf("Usage.Add() = %#v, want %#v", got, want)
	}
	if got.IsZero() {
		t.Fatal("combined usage should not be zero")
	}
}
