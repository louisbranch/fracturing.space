package aggregate

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// AssertState extracts a typed value from an any-typed state parameter.
// It accepts T, *T (dereferencing non-nil pointers), and nil (returning the zero
// value). Any other type returns a descriptive error so callers get compile-time-like
// feedback about state boundaries.
func AssertState[T any](state any) (T, error) {
	if state == nil {
		var zero T
		return zero, nil
	}
	if v, ok := state.(T); ok {
		return v, nil
	}
	if p, ok := state.(*T); ok {
		if p != nil {
			return *p, nil
		}
		var zero T
		return zero, nil
	}
	var zero T
	return zero, fmt.Errorf("expected %T, got %T", zero, state)
}

// State captures aggregate core domain state.
//
// This is the in-memory campaign-wide projection snapshot that the core decider
// uses as input. It intentionally aggregates:
// campaign-level facts,
// session lifecycle context,
// participant rosters,
// character records,
// invite records,
// and system-specific state snapshots.
//
// The struct is organized by entity maps for efficient command-time reads and
// deterministic replay folding.
type State struct {
	// Campaign is the aggregate root used for broad campaign lifecycle checks.
	Campaign campaign.State
	// Session tracks current active session, gate, and spotlight context.
	Session session.State
	// Action tracks causal roll/outcome replay state used for invariant checks.
	Action action.State
	// Participants stores compact per-participant state keyed by participant ID.
	Participants map[string]participant.State
	// Characters stores compact per-character state keyed by character ID.
	Characters map[string]character.State
	// Invites stores compact invite lifecycle state keyed by invite ID.
	Invites map[string]invite.State
	// Systems stores per-game-system runtime state keyed by system module key.
	Systems map[module.Key]any
}
