package countdowns

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestResolveCountdownAdvance(t *testing.T) {
	mutation, err := ResolveCountdownAdvance(CountdownAdvanceInput{
		Countdown: rules.Countdown{
			ID:                "  cd-1  ",
			Tone:              rules.CountdownToneProgress,
			AdvancementPolicy: rules.CountdownAdvancementPolicyManual,
			StartingValue:     6,
			RemainingValue:    4,
			LoopBehavior:      rules.CountdownLoopBehaviorNone,
			Status:            rules.CountdownStatusActive,
		},
		Amount: 2,
		Reason: "  tick  ",
	})
	if err != nil {
		t.Fatalf("ResolveCountdownAdvance() error = %v", err)
	}
	if mutation.Advance.BeforeRemaining != 4 || mutation.Advance.AfterRemaining != 2 || mutation.Advance.AdvancedBy != 2 {
		t.Fatalf("advance = %+v", mutation.Advance)
	}
	if mutation.Payload.CountdownID != "cd-1" || mutation.Payload.Reason != "tick" {
		t.Fatalf("payload = %+v", mutation.Payload)
	}
}

func TestResolveCountdownTrigger(t *testing.T) {
	result, err := ResolveCountdownTrigger(rules.Countdown{
		ID:                "cd-1",
		Tone:              rules.CountdownToneConsequence,
		AdvancementPolicy: rules.CountdownAdvancementPolicyActionDynamic,
		StartingValue:     3,
		RemainingValue:    0,
		LoopBehavior:      rules.CountdownLoopBehaviorReset,
		Status:            rules.CountdownStatusTriggerPending,
	}, "alarm")
	if err != nil {
		t.Fatalf("ResolveCountdownTrigger() error = %v", err)
	}
	if result.Payload.RemainingValueAfter != 3 || result.Payload.StatusAfter != rules.CountdownStatusActive {
		t.Fatalf("payload = %+v", result.Payload)
	}
}

func TestResolveBreathCountdownAdvance(t *testing.T) {
	success := ResolveBreathCountdownAdvance(false)
	if success.Amount != 1 || success.Reason != CountdownReasonBreathTick {
		t.Fatalf("success advance = %+v", success)
	}
	failed := ResolveBreathCountdownAdvance(true)
	if failed.Amount != 2 || failed.Reason != CountdownReasonBreathFail {
		t.Fatalf("failed advance = %+v", failed)
	}
}
