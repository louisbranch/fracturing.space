package countdowns

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

const (
	BreathCountdownName       = "Breath"
	CountdownReasonLongRest   = "long_rest"
	CountdownReasonBreathTick = "breath_tick"
	CountdownReasonBreathFail = "breath_failure"
)

type CountdownAdvanceInput struct {
	Countdown rules.Countdown
	Amount    int
	Reason    string
}

type CountdownAdvanceMutation struct {
	Advance rules.CountdownAdvance
	Payload payload.CampaignCountdownAdvancePayload
}

func ResolveCountdownAdvance(input CountdownAdvanceInput) (CountdownAdvanceMutation, error) {
	advance, err := rules.ApplyCountdownAdvance(input.Countdown, input.Amount)
	if err != nil {
		return CountdownAdvanceMutation{}, err
	}
	return CountdownAdvanceMutation{
		Advance: advance,
		Payload: payload.CampaignCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID(strings.TrimSpace(input.Countdown.ID)),
			BeforeRemaining: advance.BeforeRemaining,
			AfterRemaining:  advance.AfterRemaining,
			AdvancedBy:      advance.AdvancedBy,
			StatusBefore:    advance.StatusBefore,
			StatusAfter:     advance.StatusAfter,
			Triggered:       advance.Triggered,
			Reason:          strings.TrimSpace(input.Reason),
		},
	}, nil
}

type CountdownTriggerResolution struct {
	Result  rules.CountdownTriggerResolution
	Payload payload.CampaignCountdownTriggerResolvedPayload
}

func ResolveCountdownTrigger(countdown rules.Countdown, reason string) (CountdownTriggerResolution, error) {
	result, err := rules.ResolveCountdownTrigger(countdown)
	if err != nil {
		return CountdownTriggerResolution{}, err
	}
	return CountdownTriggerResolution{
		Result: result,
		Payload: payload.CampaignCountdownTriggerResolvedPayload{
			CountdownID:          dhids.CountdownID(strings.TrimSpace(countdown.ID)),
			StartingValueBefore:  result.StartingValueBefore,
			StartingValueAfter:   result.StartingValueAfter,
			RemainingValueBefore: result.RemainingValueBefore,
			RemainingValueAfter:  result.RemainingValueAfter,
			StatusBefore:         result.StatusBefore,
			StatusAfter:          result.StatusAfter,
			Reason:               strings.TrimSpace(reason),
		},
	}, nil
}

type BreathCountdownAdvance struct {
	Amount int
	Reason string
}

func ResolveBreathCountdownAdvance(failed bool) BreathCountdownAdvance {
	if failed {
		return BreathCountdownAdvance{Amount: 2, Reason: CountdownReasonBreathFail}
	}
	return BreathCountdownAdvance{Amount: 1, Reason: CountdownReasonBreathTick}
}
