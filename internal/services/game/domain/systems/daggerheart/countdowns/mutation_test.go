package countdowns

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestResolveCountdownMutation(t *testing.T) {
	mutation, err := ResolveCountdownMutation(CountdownMutationInput{
		Countdown: rules.Countdown{
			ID:      "  cd-1  ",
			Current: 2,
			Max:     6,
		},
		Delta:  2,
		Reason: "  tick  ",
	})
	if err != nil {
		t.Fatalf("ResolveCountdownMutation() error = %v", err)
	}
	if mutation.Update.Before != 2 || mutation.Update.After != 4 || mutation.Update.Delta != 2 {
		t.Fatalf("update = %+v, want before=2 after=4 delta=2", mutation.Update)
	}
	if mutation.Payload.CountdownID != "cd-1" {
		t.Fatalf("countdown id = %q, want cd-1", mutation.Payload.CountdownID)
	}
	if mutation.Payload.Reason != "tick" {
		t.Fatalf("reason = %q, want tick", mutation.Payload.Reason)
	}
	if mutation.Payload.Before != 2 || mutation.Payload.After != 4 || mutation.Payload.Delta != 2 {
		t.Fatalf("payload = %+v, want before=2 after=4 delta=2", mutation.Payload)
	}
}

func TestResolveCountdownMutationInvalidCountdown(t *testing.T) {
	_, err := ResolveCountdownMutation(CountdownMutationInput{
		Countdown: rules.Countdown{Current: 0, Max: 0},
		Delta:     1,
	})
	if err == nil {
		t.Fatal("expected invalid countdown error")
	}
}

func TestResolveBreathCountdownAdvance(t *testing.T) {
	success := ResolveBreathCountdownAdvance(false)
	if success.Delta != 1 || success.Reason != CountdownReasonBreathTick {
		t.Fatalf("success advance = %+v, want delta=1 reason=%q", success, CountdownReasonBreathTick)
	}

	failed := ResolveBreathCountdownAdvance(true)
	if failed.Delta != 2 || failed.Reason != CountdownReasonBreathFailure {
		t.Fatalf("failed advance = %+v, want delta=2 reason=%q", failed, CountdownReasonBreathFailure)
	}
}
