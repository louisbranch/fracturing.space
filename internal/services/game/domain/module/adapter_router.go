package module

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// AdapterRouter dispatches adapter Apply calls by event type, auto-unmarshalling
// the payload. This eliminates per-handler unmarshal boilerplate while preserving
// full handler flexibility for complex cases (read-then-write, merge logic).
type AdapterRouter struct {
	handlers map[event.Type]func(context.Context, event.Event) error
	types    []event.Type
}

// NewAdapterRouter creates an empty AdapterRouter.
func NewAdapterRouter() *AdapterRouter {
	return &AdapterRouter{
		handlers: make(map[event.Type]func(context.Context, event.Event) error),
	}
}

// Apply dispatches to the registered handler for evt.Type. Returns an error
// for unknown event types or payload unmarshal failures.
func (r *AdapterRouter) Apply(ctx context.Context, evt event.Event) error {
	handler, ok := r.handlers[evt.Type]
	if !ok {
		return fmt.Errorf("unhandled adapter event type %s", evt.Type)
	}
	return handler(ctx, evt)
}

// HandledTypes returns all registered event types in registration order.
func (r *AdapterRouter) HandledTypes() []event.Type {
	return append([]event.Type(nil), r.types...)
}

// HandleAdapter registers a typed handler for the given event type. The handler
// receives a pre-unmarshalled payload, eliminating per-case boilerplate. For
// handlers that need the raw event (e.g. to access envelope fields like
// CampaignID or Timestamp), the event.Event is also passed through.
func HandleAdapter[P any](r *AdapterRouter, t event.Type, fn func(context.Context, event.Event, P) error) {
	r.handlers[t] = func(ctx context.Context, evt event.Event) error {
		var payload P
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", t, err)
		}
		return fn(ctx, evt, payload)
	}
	r.types = append(r.types, t)
}
