package action

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

// Fold applies an event to action state.
func Fold(state State, evt event.Event) State {
	return state
}
