package fold

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Folder folds events into aggregate state.
//
// Fold applies a single event to state, returning the updated state.
// FoldHandledTypes declares which event types the Fold method handles, enabling
// startup validators to verify coverage.
//
// Named "Folder" (not "Applier") to distinguish pure state folds from
// projection.Applier, which performs side-effecting I/O writes to stores.
type Folder interface {
	Fold(state any, evt event.Event) (any, error)
	FoldHandledTypes() []event.Type
}

// CoreFoldRouter dispatches fold events to typed handler functions by event
// type for value-typed core domain state. It derives FoldHandledTypes from
// registered handlers, eliminating the sync-drift risk between a domain's
// Fold switch and its type list.
//
// S is the value type of the domain state (e.g., campaign.State). Handlers
// receive and return a value, matching the existing core domain fold signature.
//
// For system-owned event folding, see module.FoldRouter in domain/module/fold_router.go.
type CoreFoldRouter[S any] struct {
	handlers map[event.Type]func(S, event.Event) (S, error)
	types    []event.Type
}

// NewCoreFoldRouter creates a fold router for value-typed domain state.
func NewCoreFoldRouter[S any]() *CoreFoldRouter[S] {
	return &CoreFoldRouter[S]{
		handlers: make(map[event.Type]func(S, event.Event) (S, error)),
	}
}

// Handle registers a fold handler for the given event type. Panics on
// duplicate registration to surface wiring errors at init time.
func (r *CoreFoldRouter[S]) Handle(t event.Type, fn func(S, event.Event) (S, error)) {
	if _, exists := r.handlers[t]; exists {
		panic(fmt.Sprintf("duplicate fold handler for event type %s", t))
	}
	r.handlers[t] = fn
	r.types = append(r.types, t)
}

// Fold dispatches to the registered handler for the event type. Returns an
// error for unknown event types — defense-in-depth since the aggregate folder
// routes only known types.
func (r *CoreFoldRouter[S]) Fold(state S, evt event.Event) (S, error) {
	handler, ok := r.handlers[evt.Type]
	if !ok {
		return state, fmt.Errorf("unhandled fold event type: %s", evt.Type)
	}
	return handler(state, evt)
}

// FoldHandledTypes returns the event types this router handles, in
// registration order.
func (r *CoreFoldRouter[S]) FoldHandledTypes() []event.Type {
	return append([]event.Type(nil), r.types...)
}
