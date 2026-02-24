package module

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// DecideFunc handles the common unmarshal → validate → marshal → emit flow for
// simple decider cases. The validate callback receives a pointer to the
// unmarshaled payload and may mutate it (e.g. TrimSpace normalization) before
// it is re-marshaled into the emitted event. Return a non-nil Rejection to
// reject the command.
//
// The entityID callback derives the fallback entity ID from the payload when
// cmd.EntityID is empty — the common pattern in system deciders.
//
// Complex cases that emit multiple events, transform the event type, or need
// access to aggregate snapshot state should use the raw switch approach instead.
func DecideFunc[P any](
	cmd command.Command,
	eventType event.Type,
	entityType string,
	entityID func(*P) string,
	validate func(*P, func() time.Time) *command.Rejection,
	now func() time.Time,
) command.Decision {
	var payload P
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	if validate != nil {
		if rejection := validate(&payload, now); rejection != nil {
			return command.Reject(*rejection)
		}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_ENCODE_FAILED",
			Message: fmt.Sprintf("encode %s payload: %v", cmd.Type, err),
		})
	}
	eid := strings.TrimSpace(cmd.EntityID)
	if eid == "" && entityID != nil {
		eid = entityID(&payload)
	}
	etype := strings.TrimSpace(cmd.EntityType)
	if etype == "" {
		etype = entityType
	}
	evt := command.NewEvent(cmd, eventType, etype, eid, payloadJSON, now().UTC())
	return command.Accept(evt)
}

// DecideFuncTransform extends DecideFuncWithState for cases where the emitted
// event payload type differs from the command payload type. The transform
// callback converts the validated input payload into the output payload that
// will be marshaled into the emitted event.
//
// Use this when a command takes one shape (e.g. GMFearSetPayload with an
// "after" field) but the event records a different shape (e.g.
// GMFearChangedPayload with before/after/reason).
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
	var payload PIn
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	if validate != nil {
		if rejection := validate(state, hasState, &payload, now); rejection != nil {
			return command.Reject(*rejection)
		}
	}
	output := transform(state, hasState, payload)
	payloadJSON, err := json.Marshal(output)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_ENCODE_FAILED",
			Message: fmt.Sprintf("encode %s payload: %v", cmd.Type, err),
		})
	}
	eid := strings.TrimSpace(cmd.EntityID)
	if eid == "" && entityID != nil {
		eid = entityID(&payload)
	}
	etype := strings.TrimSpace(cmd.EntityType)
	if etype == "" {
		etype = entityType
	}
	evt := command.NewEvent(cmd, eventType, etype, eid, payloadJSON, now().UTC())
	return command.Accept(evt)
}

// EventSpec describes a single event to emit from a DecideFuncMulti expand
// callback. Each spec produces one event in the returned Decision.
type EventSpec struct {
	Type       event.Type
	EntityType string
	EntityID   string
	Payload    any
}

// DecideFuncMulti extends DecideFuncWithState for commands that emit multiple
// events in a single decision. The expand callback receives the validated
// payload and returns a slice of EventSpec, each of which is marshaled and
// emitted as a separate event. All events share the command's envelope fields.
//
// Use this for batch operations (e.g. multi-target damage) where the decider
// produces N events atomically from a single command.
func DecideFuncMulti[S, P any](
	cmd command.Command,
	state S,
	hasState bool,
	validate func(S, bool, *P, func() time.Time) *command.Rejection,
	expand func(S, bool, P, func() time.Time) ([]EventSpec, error),
	now func() time.Time,
) command.Decision {
	var payload P
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	if validate != nil {
		if rejection := validate(state, hasState, &payload, now); rejection != nil {
			return command.Reject(*rejection)
		}
	}
	specs, err := expand(state, hasState, payload, now)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "EXPAND_FAILED",
			Message: fmt.Sprintf("expand %s: %v", cmd.Type, err),
		})
	}
	if len(specs) == 0 {
		return command.Reject(command.Rejection{
			Code:    "NO_EVENTS",
			Message: fmt.Sprintf("expand %s produced no events", cmd.Type),
		})
	}
	ts := now().UTC()
	events := make([]event.Event, 0, len(specs))
	for _, spec := range specs {
		payloadJSON, err := json.Marshal(spec.Payload)
		if err != nil {
			return command.Reject(command.Rejection{
				Code:    "PAYLOAD_ENCODE_FAILED",
				Message: fmt.Sprintf("encode %s event payload: %v", spec.Type, err),
			})
		}
		events = append(events, command.NewEvent(cmd, spec.Type, spec.EntityType, spec.EntityID, payloadJSON, ts))
	}
	return command.Accept(events...)
}

// DecideFuncWithState extends DecideFunc with typed snapshot state for
// pre-validation. The caller provides the already-extracted state and whether
// the extraction succeeded (hasState). The validate callback can use the state
// to enforce before-value checks, idempotency guards, or snapshot-based
// invariants before the payload is marshaled into the emitted event.
//
// Use this when a decider case needs snapshot access but otherwise follows the
// standard unmarshal → validate → marshal → emit flow.
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
	var payload P
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	if validate != nil {
		if rejection := validate(state, hasState, &payload, now); rejection != nil {
			return command.Reject(*rejection)
		}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_ENCODE_FAILED",
			Message: fmt.Sprintf("encode %s payload: %v", cmd.Type, err),
		})
	}
	eid := strings.TrimSpace(cmd.EntityID)
	if eid == "" && entityID != nil {
		eid = entityID(&payload)
	}
	etype := strings.TrimSpace(cmd.EntityType)
	if etype == "" {
		etype = entityType
	}
	evt := command.NewEvent(cmd, eventType, etype, eid, payloadJSON, now().UTC())
	return command.Accept(evt)
}
