package daggerheart

import "testing"

func TestResolveRestOutcomeShortInterrupted(t *testing.T) {
	state := RestState{}
	outcome, err := ResolveRestOutcome(state, RestTypeShort, true, 1, 3)
	if err != nil {
		t.Fatalf("ResolveRestOutcome returned error: %v", err)
	}
	if outcome.Applied {
		t.Fatal("expected interrupted short rest to not apply")
	}
}

func TestResolveRestOutcomeShortIncrements(t *testing.T) {
	state := RestState{}
	outcome, err := ResolveRestOutcome(state, RestTypeShort, false, 1, 3)
	if err != nil {
		t.Fatalf("ResolveRestOutcome returned error: %v", err)
	}
	if outcome.State.ConsecutiveShortRests != 1 {
		t.Fatalf("short rests = %d, want 1", outcome.State.ConsecutiveShortRests)
	}
	if outcome.GMFearGain == 0 {
		t.Fatal("expected GM fear gain")
	}
	if outcome.AdvanceCountdown {
		t.Fatal("expected no countdown advance on short rest")
	}
	if !outcome.RefreshRest || outcome.RefreshLongRest {
		t.Fatal("expected short rest to refresh rest-only effects")
	}
}

func TestResolveRestOutcomeLongResets(t *testing.T) {
	state := RestState{ConsecutiveShortRests: 2}
	outcome, err := ResolveRestOutcome(state, RestTypeLong, false, 2, 2)
	if err != nil {
		t.Fatalf("ResolveRestOutcome returned error: %v", err)
	}
	if outcome.State.ConsecutiveShortRests != 0 {
		t.Fatalf("short rests = %d, want 0", outcome.State.ConsecutiveShortRests)
	}
	if outcome.GMFearGain == 0 {
		t.Fatal("expected GM fear gain")
	}
	if !outcome.AdvanceCountdown {
		t.Fatal("expected countdown advance on long rest")
	}
	if !outcome.RefreshRest || !outcome.RefreshLongRest {
		t.Fatal("expected long rest to refresh all effects")
	}
}

func TestResolveRestOutcomeLongInterruptedUsesShort(t *testing.T) {
	state := RestState{ConsecutiveShortRests: 1}
	outcome, err := ResolveRestOutcome(state, RestTypeLong, true, 3, 2)
	if err != nil {
		t.Fatalf("ResolveRestOutcome returned error: %v", err)
	}
	if outcome.EffectiveType != RestTypeShort {
		t.Fatalf("effective rest = %v, want short", outcome.EffectiveType)
	}
	if outcome.State.ConsecutiveShortRests != 2 {
		t.Fatalf("short rests = %d, want 2", outcome.State.ConsecutiveShortRests)
	}
}

func TestResolveRestOutcomeRejectsFourthShort(t *testing.T) {
	state := RestState{ConsecutiveShortRests: 3}
	_, err := ResolveRestOutcome(state, RestTypeShort, false, 4, 3)
	if err == nil {
		t.Fatal("expected error for fourth short rest")
	}
}
