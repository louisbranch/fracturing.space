package countdowns

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

const (
	// BreathCountdownName is the canonical countdown label for underwater breath.
	BreathCountdownName = "Breath"
	// BreathCountdownInitial is the canonical initial breath countdown value.
	BreathCountdownInitial = 0
	// BreathCountdownMax is the canonical breath countdown max value.
	BreathCountdownMax = 3
	// CountdownReasonLongRest identifies a long-rest-triggered countdown advance.
	CountdownReasonLongRest = "long_rest"
	// CountdownReasonBreathTick identifies a successful breath countdown tick.
	CountdownReasonBreathTick = "breath_tick"
	// CountdownReasonBreathFailure identifies a failed breath countdown tick.
	CountdownReasonBreathFailure = "breath_failure"
)

// CountdownMutationInput captures transport-agnostic countdown mutation input.
type CountdownMutationInput struct {
	Countdown rules.Countdown
	Delta     int
	Override  *int
	Reason    string
}

// CountdownMutation captures resolved countdown update state and event payload.
type CountdownMutation struct {
	Update  rules.CountdownUpdate
	Payload payload.CountdownUpdatePayload
}

// ResolveCountdownMutation applies countdown rules and builds a canonical
// countdown update payload for command/event emission.
func ResolveCountdownMutation(input CountdownMutationInput) (CountdownMutation, error) {
	update, err := rules.ApplyCountdownUpdate(input.Countdown, input.Delta, input.Override)
	if err != nil {
		return CountdownMutation{}, err
	}
	return CountdownMutation{
		Update: update,
		Payload: payload.CountdownUpdatePayload{
			CountdownID: ids.CountdownID(strings.TrimSpace(input.Countdown.ID)),
			Before:      update.Before,
			After:       update.After,
			Delta:       update.Delta,
			Looped:      update.Looped,
			Reason:      strings.TrimSpace(input.Reason),
		},
	}, nil
}

// BreathCountdownAdvance captures the countdown delta/reason for one breath
// advancement step.
type BreathCountdownAdvance struct {
	Delta  int
	Reason string
}

// ResolveBreathCountdownAdvance returns the canonical mutation for a breath
// countdown advance.
func ResolveBreathCountdownAdvance(failed bool) BreathCountdownAdvance {
	if failed {
		return BreathCountdownAdvance{Delta: 2, Reason: CountdownReasonBreathFailure}
	}
	return BreathCountdownAdvance{Delta: 1, Reason: CountdownReasonBreathTick}
}
