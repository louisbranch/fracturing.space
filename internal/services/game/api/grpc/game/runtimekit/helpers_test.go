package runtimekit

import (
	"testing"
	"time"
)

func TestFixedClockReturnsSameTime(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := FixedClock(now)
	if got := clock(); !got.Equal(now) {
		t.Fatalf("FixedClock() = %v, want %v", got, now)
	}
	if got := clock(); !got.Equal(now) {
		t.Fatalf("FixedClock() second call = %v, want %v", got, now)
	}
}

func TestFixedIDGeneratorAlwaysReturnsSameID(t *testing.T) {
	generator := FixedIDGenerator("fixed-id")
	for i := 0; i < 3; i++ {
		got, err := generator()
		if err != nil {
			t.Fatalf("FixedIDGenerator() error = %v", err)
		}
		if got != "fixed-id" {
			t.Fatalf("FixedIDGenerator() = %q, want %q", got, "fixed-id")
		}
	}
}

func TestFixedSequenceIDGeneratorReturnsSequenceThenRepeatsLast(t *testing.T) {
	generator := FixedSequenceIDGenerator("id-1", "id-2")
	want := []string{"id-1", "id-2", "id-2", "id-2"}
	for i, expected := range want {
		got, err := generator()
		if err != nil {
			t.Fatalf("FixedSequenceIDGenerator() call %d error = %v", i, err)
		}
		if got != expected {
			t.Fatalf("FixedSequenceIDGenerator() call %d = %q, want %q", i, got, expected)
		}
	}
}

func TestFixedSequenceIDGeneratorPanicsWithoutIDs(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected FixedSequenceIDGenerator() to panic without ids")
		}
	}()
	_ = FixedSequenceIDGenerator()
}

func TestSequentialIDGeneratorUsesDecimalSuffixes(t *testing.T) {
	generator := SequentialIDGenerator("item")
	var got []string
	for i := 0; i < 12; i++ {
		id, err := generator()
		if err != nil {
			t.Fatalf("SequentialIDGenerator() call %d error = %v", i, err)
		}
		got = append(got, id)
	}
	want := []string{
		"item-1",
		"item-2",
		"item-3",
		"item-4",
		"item-5",
		"item-6",
		"item-7",
		"item-8",
		"item-9",
		"item-10",
		"item-11",
		"item-12",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SequentialIDGenerator() call %d = %q, want %q", i, got[i], want[i])
		}
	}
}
