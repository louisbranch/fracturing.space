package module

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldRouter dispatches fold events to typed handler functions by event type.
// It eliminates the per-case unmarshal boilerplate that system folders
// otherwise repeat for every event type.
//
// S must be a pointer type so handlers can mutate state in place. The assert
// callback converts the untyped state (any) to *S, handling nil initialization
// when the state is first created.
type FoldRouter[S any] struct {
	assert   func(any) (S, error)
	handlers map[event.Type]func(S, event.Event) error
	types    []event.Type
}

// NewFoldRouter creates a fold router. The assert callback converts untyped
// state to the system's concrete state pointer, returning an error on type
// mismatch. It must handle nil input by creating a zero-value state.
func NewFoldRouter[S any](assert func(any) (S, error)) *FoldRouter[S] {
	return &FoldRouter[S]{
		assert:   assert,
		handlers: make(map[event.Type]func(S, event.Event) error),
	}
}

// Fold asserts the state type, dispatches to the registered handler for the
// event type, and returns the (possibly mutated) state. Unknown event types
// return an error, matching the behavior of the manual default: case in
// hand-written fold switches.
func (r *FoldRouter[S]) Fold(state any, evt event.Event) (any, error) {
	s, err := r.assert(state)
	if err != nil {
		return nil, err
	}
	handler, ok := r.handlers[evt.Type]
	if !ok {
		return nil, fmt.Errorf("unhandled fold event type: %s", evt.Type)
	}
	if err := handler(s, evt); err != nil {
		return nil, err
	}
	return s, nil
}

// FoldHandledTypes returns the event types this router handles, in
// registration order. Used by ValidateSystemFoldCoverage to verify at startup
// that every emittable event type with replay intent has a fold handler.
func (r *FoldRouter[S]) FoldHandledTypes() []event.Type {
	return append([]event.Type(nil), r.types...)
}

// HandleFold registers a typed fold handler for the given event type. The
// handler receives the already-asserted state pointer and a typed payload
// auto-unmarshaled from the event's PayloadJSON.
//
// This is a top-level generic function because Go disallows method-level type
// parameters on generic types.
func HandleFold[S, P any](r *FoldRouter[S], t event.Type, fn func(S, P) error) {
	r.handlers[t] = func(s S, evt event.Event) error {
		var payload P
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", evt.Type, err)
		}
		return fn(s, payload)
	}
	r.types = append(r.types, t)
}
