package participant

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers participant commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeJoin,
		Owner:           command.OwnerCore,
		ValidatePayload: validateJoinPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeUpdate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateUpdatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeLeave,
		Owner:           command.OwnerCore,
		ValidatePayload: validateLeavePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeBind,
		Owner:           command.OwnerCore,
		ValidatePayload: validateBindPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeUnbind,
		Owner:           command.OwnerCore,
		ValidatePayload: validateUnbindPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeSeatReassignLegacy,
		Owner:           command.OwnerCore,
		ValidatePayload: validateSeatReassignPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeSeatReassign,
		Owner:           command.OwnerCore,
		ValidatePayload: validateSeatReassignPayload,
	}); err != nil {
		return err
	}
	return nil
}

// EmittableEventTypes returns all event types the participant decider can emit.
func EmittableEventTypes() []event.Type {
	return []event.Type{
		EventTypeJoined,
		EventTypeUpdated,
		EventTypeLeft,
		EventTypeBound,
		EventTypeUnbound,
		EventTypeSeatReassigned,
	}
}

// RegisterEvents registers participant events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeJoined,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateJoinPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeUpdated,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateUpdatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeLeft,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateLeavePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeBound,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateBindPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeUnbound,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateUnbindPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeSeatReassignedLegacy,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateSeatReassignPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeSeatReassigned,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateSeatReassignPayload,
	}); err != nil {
		return err
	}
	return nil
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
