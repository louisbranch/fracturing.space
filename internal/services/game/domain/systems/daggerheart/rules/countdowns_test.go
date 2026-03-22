package rules

import "testing"

func TestNormalizeCountdownTone(t *testing.T) {
	tests := []struct {
		value   string
		want    string
		wantErr bool
	}{
		{" progress ", CountdownToneProgress, false},
		{"CONSEQUENCE", CountdownToneConsequence, false},
		{"neutral", CountdownToneNeutral, false},
		{"", "", true},
		{"other", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizeCountdownTone(tt.value)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeCountdownTone(%q) expected error", tt.value)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeCountdownTone(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeCountdownTone(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestApplyCountdownAdvance(t *testing.T) {
	countdown := Countdown{
		Tone:              CountdownToneProgress,
		AdvancementPolicy: CountdownAdvancementPolicyManual,
		StartingValue:     4,
		RemainingValue:    3,
		LoopBehavior:      CountdownLoopBehaviorReset,
		Status:            CountdownStatusActive,
	}
	update, err := ApplyCountdownAdvance(countdown, 2)
	if err != nil {
		t.Fatalf("ApplyCountdownAdvance() error = %v", err)
	}
	if update.BeforeRemaining != 3 || update.AfterRemaining != 1 || update.AdvancedBy != 2 {
		t.Fatalf("unexpected update = %+v", update)
	}
	if update.StatusAfter != CountdownStatusActive || update.Triggered {
		t.Fatalf("unexpected trigger status = %+v", update)
	}
}

func TestApplyCountdownAdvanceTriggersPending(t *testing.T) {
	countdown := Countdown{
		Tone:              CountdownToneConsequence,
		AdvancementPolicy: CountdownAdvancementPolicyActionDynamic,
		StartingValue:     3,
		RemainingValue:    1,
		LoopBehavior:      CountdownLoopBehaviorResetIncreaseStart,
		Status:            CountdownStatusActive,
	}
	update, err := ApplyCountdownAdvance(countdown, 2)
	if err != nil {
		t.Fatalf("ApplyCountdownAdvance() error = %v", err)
	}
	if !update.Triggered || update.AfterRemaining != 0 || update.StatusAfter != CountdownStatusTriggerPending {
		t.Fatalf("unexpected trigger update = %+v", update)
	}
}

func TestResolveCountdownTrigger(t *testing.T) {
	countdown := Countdown{
		Tone:              CountdownToneProgress,
		AdvancementPolicy: CountdownAdvancementPolicyActionDynamic,
		StartingValue:     3,
		RemainingValue:    0,
		LoopBehavior:      CountdownLoopBehaviorResetIncreaseStart,
		Status:            CountdownStatusTriggerPending,
	}
	result, err := ResolveCountdownTrigger(countdown)
	if err != nil {
		t.Fatalf("ResolveCountdownTrigger() error = %v", err)
	}
	if result.StartingValueAfter != 4 || result.RemainingValueAfter != 4 || result.StatusAfter != CountdownStatusActive {
		t.Fatalf("unexpected trigger resolution = %+v", result)
	}
}

func TestDynamicCountdownAdvanceAmount(t *testing.T) {
	tests := []struct {
		tone    string
		outcome string
		want    int
	}{
		{CountdownToneProgress, "failure_with_fear", 0},
		{CountdownToneConsequence, "failure_with_fear", 3},
		{CountdownToneProgress, "success_with_hope", 2},
		{CountdownToneConsequence, "success_with_hope", 0},
		{CountdownToneProgress, "critical_success", 3},
	}
	for _, tt := range tests {
		got, err := DynamicCountdownAdvanceAmount(tt.tone, tt.outcome)
		if err != nil {
			t.Fatalf("DynamicCountdownAdvanceAmount(%q, %q) error = %v", tt.tone, tt.outcome, err)
		}
		if got != tt.want {
			t.Fatalf("DynamicCountdownAdvanceAmount(%q, %q) = %d, want %d", tt.tone, tt.outcome, got, tt.want)
		}
	}
}
