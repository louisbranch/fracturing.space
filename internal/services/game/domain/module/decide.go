package module

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/decide"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// EventSpec describes a single event to emit from a DecideFuncMulti expand
// callback. Each spec produces one event in the returned Decision.
type EventSpec = decide.EventSpec

// DecideFunc handles the common unmarshal -> validate -> marshal -> emit flow for
// simple decider cases.
func DecideFunc[P any](
	cmd command.Command,
	eventType event.Type,
	entityType string,
	entityID func(*P) string,
	validate func(*P, func() time.Time) *command.Rejection,
	now func() time.Time,
) command.Decision {
	return decide.DecideFunc(cmd, eventType, entityType, entityID, validate, now)
}

// DecideFuncTransform handles decider cases where output event payload shape
// differs from command payload shape.
func DecideFuncTransform[S, PIn, POut any](
	cmd command.Command,
	state S,
	hasState bool,
	eventType event.Type,
	entityType string,
	entityID func(*PIn) string,
	validate func(S, bool, *PIn, func() time.Time) *command.Rejection,
	transform func(S, bool, PIn) POut,
	now func() time.Time,
) command.Decision {
	return decide.DecideFuncTransform(cmd, state, hasState, eventType, entityType, entityID, validate, transform, now)
}

// DecideFuncMulti handles decider cases that emit multiple events for one
// validated command payload.
func DecideFuncMulti[S, P any](
	cmd command.Command,
	state S,
	hasState bool,
	validate func(S, bool, *P, func() time.Time) *command.Rejection,
	expand func(S, bool, P, func() time.Time) ([]EventSpec, error),
	now func() time.Time,
) command.Decision {
	return decide.DecideFuncMulti(cmd, state, hasState, validate, expand, now)
}

// DecideFuncWithState handles typed-state decider cases that still follow the
// standard unmarshal -> validate -> marshal -> emit shape.
func DecideFuncWithState[S, P any](
	cmd command.Command,
	state S,
	hasState bool,
	eventType event.Type,
	entityType string,
	entityID func(*P) string,
	validate func(S, bool, *P, func() time.Time) *command.Rejection,
	now func() time.Time,
) command.Decision {
	return decide.DecideFuncWithState(cmd, state, hasState, eventType, entityType, entityID, validate, now)
}
