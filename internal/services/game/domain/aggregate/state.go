package aggregate

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// AssertState extracts a typed value from an any-typed state parameter.
// It accepts T and *T (dereferencing non-nil pointers). Nil inputs return an
// error because nil state reaching a fold or decider indicates a missing
// StateFactory or broken replay pipeline — failing loud surfaces the root cause
// early instead of silently operating on zero-valued state with nil maps.
func AssertState[T any](state any) (T, error) {
	if state == nil {
		var zero T
		return zero, fmt.Errorf("expected %T, got nil (missing StateFactory?)", zero)
	}
	if v, ok := state.(T); ok {
		return v, nil
	}
	if p, ok := state.(*T); ok {
		if p == nil {
			var zero T
			return zero, fmt.Errorf("expected %T, got nil pointer (missing StateFactory?)", zero)
		}
		return *p, nil
	}
	var zero T
	return zero, fmt.Errorf("expected %T, got %T", zero, state)
}

// NewState returns a State with all entity maps initialized to avoid nil-map
// panics during fold operations or direct writes.
func NewState() State {
	return State{
		Participants: make(map[ids.ParticipantID]participant.State),
		Characters:   make(map[ids.CharacterID]character.State),
		Invites:      make(map[ids.InviteID]invite.State),
		Scenes:       make(map[ids.SceneID]scene.State),
		Systems:      make(map[module.Key]any),
	}
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
	Participants map[ids.ParticipantID]participant.State
	// Characters stores compact per-character state keyed by character ID.
	Characters map[ids.CharacterID]character.State
	// Invites stores compact invite lifecycle state keyed by invite ID.
	Invites map[ids.InviteID]invite.State
	// Scenes stores per-scene state keyed by scene ID.
	Scenes map[ids.SceneID]scene.State
	// Systems stores per-game-system runtime state keyed by system module key.
	Systems map[module.Key]any
}
