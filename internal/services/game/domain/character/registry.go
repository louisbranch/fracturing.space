package character

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers character commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeCreate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateCreatePayload,
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
		Type:            commandTypeDelete,
		Owner:           command.OwnerCore,
		ValidatePayload: validateDeletePayload,
	}); err != nil {
		return err
	}
	return registry.Register(command.Definition{
		Type:            commandTypeProfileUpdate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateProfileUpdatePayload,
	})
}

// RegisterEvents registers character events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeCreated,
		Owner:           event.OwnerCore,
		ValidatePayload: validateCreatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeUpdated,
		Owner:           event.OwnerCore,
		ValidatePayload: validateUpdatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeDeleted,
		Owner:           event.OwnerCore,
		ValidatePayload: validateDeletePayload,
	}); err != nil {
		return err
	}
	return registry.Register(event.Definition{
		Type:            eventTypeProfileUpdated,
		Owner:           event.OwnerCore,
		ValidatePayload: validateProfileUpdatePayload,
	})
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
