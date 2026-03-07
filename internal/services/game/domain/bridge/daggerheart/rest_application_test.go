package daggerheart

import "testing"

func TestResolveRestApplication_LongRestWithCountdown(t *testing.T) {
	payload, err := ResolveRestApplication(RestApplicationInput{
		RestType:              RestTypeLong,
		Interrupted:           false,
		CurrentGMFear:         GMFearMax - 1,
		ConsecutiveShortRests: 2,
		Outcome: RestOutcome{
			GMFearGain:       4,
			AdvanceCountdown: true,
			RefreshRest:      true,
			RefreshLongRest:  true,
			State:            RestState{ConsecutiveShortRests: 0},
		},
		CharacterIDs: []string{" char-1 ", "", "char-2"},
		LongTermCountdownState: &Countdown{
			ID:      "cd-1",
			Current: 2,
			Max:     6,
		},
	})
	if err != nil {
		t.Fatalf("ResolveRestApplication() error = %v", err)
	}
	if payload.RestType != "long" {
		t.Fatalf("rest type = %q, want long", payload.RestType)
	}
	if payload.GMFearBefore != GMFearMax-1 || payload.GMFearAfter != GMFearMax {
		t.Fatalf("gm fear = %d->%d, want %d->%d", payload.GMFearBefore, payload.GMFearAfter, GMFearMax-1, GMFearMax)
	}
	if payload.ShortRestsBefore != 2 || payload.ShortRestsAfter != 0 {
		t.Fatalf("short rests = %d->%d, want 2->0", payload.ShortRestsBefore, payload.ShortRestsAfter)
	}
	if payload.LongTermCountdown == nil {
		t.Fatal("expected long term countdown payload")
	}
	if payload.LongTermCountdown.CountdownID != "cd-1" {
		t.Fatalf("countdown id = %q, want cd-1", payload.LongTermCountdown.CountdownID)
	}
	if payload.LongTermCountdown.Before != 2 || payload.LongTermCountdown.After != 3 || payload.LongTermCountdown.Delta != 1 {
		t.Fatalf("long term countdown = %+v, want before=2 after=3 delta=1", payload.LongTermCountdown)
	}
	if payload.LongTermCountdown.Reason != CountdownReasonLongRest {
		t.Fatalf("countdown reason = %q, want %q", payload.LongTermCountdown.Reason, CountdownReasonLongRest)
	}
	if len(payload.CharacterStates) != 2 {
		t.Fatalf("character states len = %d, want 2", len(payload.CharacterStates))
	}
	if payload.CharacterStates[0].CharacterID != "char-1" || payload.CharacterStates[1].CharacterID != "char-2" {
		t.Fatalf("character ids = %+v, want [char-1 char-2]", payload.CharacterStates)
	}
}

func TestResolveRestApplication_AdvanceWithoutCountdownState(t *testing.T) {
	payload, err := ResolveRestApplication(RestApplicationInput{
		RestType:              RestTypeShort,
		Interrupted:           true,
		CurrentGMFear:         2,
		ConsecutiveShortRests: 1,
		Outcome: RestOutcome{
			GMFearGain:       1,
			AdvanceCountdown: true,
			RefreshRest:      false,
			RefreshLongRest:  false,
			State:            RestState{ConsecutiveShortRests: 1},
		},
	})
	if err != nil {
		t.Fatalf("ResolveRestApplication() error = %v", err)
	}
	if payload.LongTermCountdown != nil {
		t.Fatal("expected no long term countdown payload")
	}
	if payload.RestType != "short" {
		t.Fatalf("rest type = %q, want short", payload.RestType)
	}
}

func TestResolveRestApplication_InvalidCountdownState(t *testing.T) {
	_, err := ResolveRestApplication(RestApplicationInput{
		RestType:              RestTypeLong,
		Interrupted:           false,
		CurrentGMFear:         0,
		ConsecutiveShortRests: 0,
		Outcome: RestOutcome{
			AdvanceCountdown: true,
			State:            RestState{},
		},
		LongTermCountdownState: &Countdown{
			ID:      "cd-invalid",
			Current: 0,
			Max:     0,
		},
	})
	if err == nil {
		t.Fatal("expected invalid countdown state error")
	}
}
