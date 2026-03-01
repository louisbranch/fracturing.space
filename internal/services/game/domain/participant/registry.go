package participant

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type commandRegistration struct {
	definition command.Definition
}

type eventProjectionRegistration struct {
	definition event.Definition
	emittable  bool
	projection bool
}

var participantCommandRegistrations = []commandRegistration{
	{
		definition: command.Definition{
			Type:            CommandTypeJoin,
			Owner:           command.OwnerCore,
			ValidatePayload: validateJoinPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeUpdate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateUpdatePayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeLeave,
			Owner:           command.OwnerCore,
			ValidatePayload: validateLeavePayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeBind,
			Owner:           command.OwnerCore,
			ValidatePayload: validateBindPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeUnbind,
			Owner:           command.OwnerCore,
			ValidatePayload: validateUnbindPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSeatReassign,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSeatReassignPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSeatReassignLegacy,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSeatReassignPayload,
		},
	},
}

var participantEventRegistrations = []eventProjectionRegistration{
	{
		definition: event.Definition{
			Type:            EventTypeJoined,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateJoinPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeUpdated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateUpdatePayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeLeft,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateLeavePayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeBound,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateBindPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeUnbound,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateUnbindPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeSeatReassignedLegacy,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSeatReassignPayload,
		},
		emittable:  false,
		projection: false,
	},
	{
		definition: event.Definition{
			Type:            EventTypeSeatReassigned,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSeatReassignPayload,
		},
		emittable:  true,
		projection: true,
	},
}

// RegisterCommands registers participant commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, registration := range participantCommandRegistrations {
		if err := registry.Register(registration.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the participant decider can emit.
func EmittableEventTypes() []event.Type {
	return participantEventTypes(func(registration eventProjectionRegistration) bool {
		return registration.emittable
	})
}

// DeciderHandledCommands returns all command types the participant decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(participantCommandRegistrations))
	for _, registration := range participantCommandRegistrations {
		types = append(types, registration.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the participant event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return participantEventTypes(func(registration eventProjectionRegistration) bool {
		return registration.projection
	})
}

// RegisterEvents registers participant events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, registration := range participantEventRegistrations {
		if err := registry.Register(registration.definition); err != nil {
			return err
		}
	}
	return nil
}

func participantEventTypes(include func(eventProjectionRegistration) bool) []event.Type {
	types := make([]event.Type, 0, len(participantEventRegistrations))
	for _, registration := range participantEventRegistrations {
		if include(registration) {
			types = append(types, registration.definition.Type)
		}
	}
	return types
}

// validateJoinPayload ensures join payloads match the participant join shape.
func validateJoinPayload(raw json.RawMessage) error {
	var payload JoinPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateUpdatePayload ensures update payloads match the participant update shape.
func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateLeavePayload ensures leave payloads match the participant leave shape.
func validateLeavePayload(raw json.RawMessage) error {
	var payload LeavePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateBindPayload ensures bind payloads match the participant bind shape.
func validateBindPayload(raw json.RawMessage) error {
	var payload BindPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateUnbindPayload ensures unbind payloads match the participant unbind shape.
func validateUnbindPayload(raw json.RawMessage) error {
	var payload UnbindPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateSeatReassignPayload ensures reassign payloads match the seat reassign shape.
func validateSeatReassignPayload(raw json.RawMessage) error {
	var payload SeatReassignPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}
