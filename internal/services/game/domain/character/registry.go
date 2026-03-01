package character

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type commandContract struct {
	definition command.Definition
}

type eventProjectionContract struct {
	definition event.Definition
	emittable  bool
	projection bool
}

var characterCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeCreate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCreatePayload,
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
			Type:            CommandTypeDelete,
			Owner:           command.OwnerCore,
			ValidatePayload: validateDeletePayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeProfileUpdate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateProfileUpdatePayload,
		},
	},
}

var characterEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeCreated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCreatePayload,
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
			Type:            EventTypeDeleted,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateDeletePayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeProfileUpdated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateProfileUpdatePayload,
		},
		emittable:  true,
		projection: true,
	},
}

// RegisterCommands registers character commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range characterCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the character decider can emit.
func EmittableEventTypes() []event.Type {
	return characterEventTypes(func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the character decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(characterCommandContracts))
	for _, contract := range characterCommandContracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the character event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return characterEventTypes(func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

// RegisterEvents registers character events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range characterEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

func characterEventTypes(include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(characterEventContracts))
	for _, contract := range characterEventContracts {
		if include(contract) {
			types = append(types, contract.definition.Type)
		}
	}
	return types
}

// validateCreatePayload ensures create payloads match the character create shape.
func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateUpdatePayload ensures update payloads match the character update shape.
func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateDeletePayload ensures delete payloads match the character delete shape.
func validateDeletePayload(raw json.RawMessage) error {
	var payload DeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateProfileUpdatePayload ensures profile update payloads match the profile update shape.
func validateProfileUpdatePayload(raw json.RawMessage) error {
	var payload ProfileUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}
