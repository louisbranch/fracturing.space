package projection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// CoreRouter dispatches core projection events by type, checking store and ID
// preconditions before calling the handler. Typed handlers registered via
// HandleProjection receive auto-unmarshalled payloads, eliminating the
// per-handler decodePayload boilerplate.
type CoreRouter struct {
	handlers map[event.Type]coreHandlerEntry
	types    []event.Type
}

// coreHandlerEntry declares the preconditions and apply function for one event
// type. This is the router-internal equivalent of the former handlerEntry.
type coreHandlerEntry struct {
	stores storeRequirement
	ids    idRequirement
	apply  func(Applier, context.Context, event.Event) error
}

// NewCoreRouter creates an empty CoreRouter.
func NewCoreRouter() *CoreRouter {
	return &CoreRouter{
		handlers: make(map[event.Type]coreHandlerEntry),
	}
}

// Route dispatches an event to the registered handler after checking store and
// ID preconditions. Returns an error for unknown event types, precondition
// failures, or handler errors.
func (r *CoreRouter) Route(a Applier, ctx context.Context, evt event.Event) error {
	h, ok := r.handlers[evt.Type]
	if !ok {
		return fmt.Errorf("unhandled projection event type: %s", evt.Type)
	}
	if err := a.validatePreconditions(handlerEntry{stores: h.stores, ids: h.ids}, evt); err != nil {
		return err
	}
	return h.apply(a, ctx, evt)
}

// HandledTypes returns all registered event types in registration order.
func (r *CoreRouter) HandledTypes() []event.Type {
	return append([]event.Type(nil), r.types...)
}

// HandleProjection registers a typed handler for the given event type. The
// handler receives a pre-unmarshalled payload, eliminating per-case
// decodePayload boilerplate. The event.Event is also passed through for
// envelope fields (CampaignID, EntityID, Timestamp, etc.).
func HandleProjection[P any](r *CoreRouter, t event.Type, stores storeRequirement, ids idRequirement,
	fn func(Applier, context.Context, event.Event, P) error) {
	r.handlers[t] = coreHandlerEntry{
		stores: stores,
		ids:    ids,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			var payload P
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				return fmt.Errorf("decode %s payload: %w", t, err)
			}
			return fn(a, ctx, evt, payload)
		},
	}
	r.types = append(r.types, t)
}

// HandleProjectionRaw registers a handler that does not unmarshal a payload.
// Use for event types where the handler needs no payload data (e.g.
// spotlight_cleared).
func HandleProjectionRaw(r *CoreRouter, t event.Type, stores storeRequirement, ids idRequirement,
	fn func(Applier, context.Context, event.Event) error) {
	r.handlers[t] = coreHandlerEntry{
		stores: stores,
		ids:    ids,
		apply:  fn,
	}
	r.types = append(r.types, t)
}
