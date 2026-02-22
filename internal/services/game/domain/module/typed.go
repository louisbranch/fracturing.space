package module

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// TypedProjector wraps a typed fold function to satisfy the Projector interface.
// System authors provide strongly-typed fold logic; the wrapper handles the
// any → S assertion so callers never see raw type switches.
type TypedProjector[S any] struct {
	// Assert converts the untyped state to S, returning an error on mismatch.
	Assert func(any) (S, error)
	// Fold applies an event to the typed state.
	Fold func(S, event.Event) (S, error)
	// Types declares which event types this projector handles.
	Types func() []event.Type
}

// Apply satisfies Projector by asserting state to S, folding, and returning as any.
func (p TypedProjector[S]) Apply(state any, evt event.Event) (any, error) {
	if p.Assert == nil {
		return nil, fmt.Errorf("typed projector: Assert function is nil")
	}
	if p.Fold == nil {
		return nil, fmt.Errorf("typed projector: Fold function is nil")
	}
	s, err := p.Assert(state)
	if err != nil {
		return nil, err
	}
	return p.Fold(s, evt)
}

// FoldHandledTypes satisfies Projector by delegating to the Types function.
func (p TypedProjector[S]) FoldHandledTypes() []event.Type {
	if p.Types == nil {
		return nil
	}
	return p.Types()
}

// TypedDecider wraps a typed decide function to satisfy the Decider interface.
// System authors provide strongly-typed decision logic; the wrapper handles the
// any → S assertion so callers never see raw type switches.
type TypedDecider[S any] struct {
	// Assert converts the untyped state to S, returning an error on mismatch.
	Assert func(any) (S, error)
	// Fn contains the typed decision logic.
	Fn func(S, command.Command, func() time.Time) command.Decision
}

// Decide satisfies Decider by asserting state to S and delegating to Fn.
func (d TypedDecider[S]) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	if d.Assert == nil {
		return command.Reject(command.Rejection{
			Code:    "STATE_ASSERT_FAILED",
			Message: "typed decider: Assert function is nil",
		})
	}
	if d.Fn == nil {
		return command.Reject(command.Rejection{
			Code:    "STATE_ASSERT_FAILED",
			Message: "typed decider: Fn function is nil",
		})
	}
	s, err := d.Assert(state)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "STATE_ASSERT_FAILED",
			Message: fmt.Sprintf("typed decider state assertion: %v", err),
		})
	}
	return d.Fn(s, cmd, now)
}
