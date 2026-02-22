package invite

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers invite commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	if err := registry.Register(command.Definition{
		Type:            CommandTypeCreate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateCreatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            CommandTypeClaim,
		Owner:           command.OwnerCore,
		ValidatePayload: validateClaimPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            CommandTypeRevoke,
		Owner:           command.OwnerCore,
		ValidatePayload: validateRevokePayload,
	}); err != nil {
		return err
	}
	return registry.Register(command.Definition{
		Type:            CommandTypeUpdate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateUpdatePayload,
	})
}

// EmittableEventTypes returns all event types the invite decider can emit.
func EmittableEventTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeClaimed,
		EventTypeRevoked,
		EventTypeUpdated,
	}
}

// DeciderHandledCommands returns all command types the invite decider handles.
func DeciderHandledCommands() []command.Type {
	return []command.Type{
		CommandTypeCreate,
		CommandTypeClaim,
		CommandTypeRevoke,
		CommandTypeUpdate,
	}
}

// ProjectionHandledTypes returns the invite event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeClaimed,
		EventTypeRevoked,
		EventTypeUpdated,
	}
}

// RegisterEvents registers invite events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeCreated,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateCreatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeClaimed,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateClaimPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            EventTypeRevoked,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateRevokePayload,
	}); err != nil {
		return err
	}
	return registry.Register(event.Definition{
		Type:            EventTypeUpdated,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateUpdatePayload,
	})
}

func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	return json.Unmarshal(raw, &payload)
}

func validateClaimPayload(raw json.RawMessage) error {
	var payload ClaimPayload
	return json.Unmarshal(raw, &payload)
}

func validateRevokePayload(raw json.RawMessage) error {
	var payload RevokePayload
	return json.Unmarshal(raw, &payload)
}

func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	return json.Unmarshal(raw, &payload)
}
