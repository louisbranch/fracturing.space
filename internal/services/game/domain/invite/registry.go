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
		Type:            commandTypeCreate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateCreatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeClaim,
		Owner:           command.OwnerCore,
		ValidatePayload: validateClaimPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(command.Definition{
		Type:            commandTypeRevoke,
		Owner:           command.OwnerCore,
		ValidatePayload: validateRevokePayload,
	}); err != nil {
		return err
	}
	return registry.Register(command.Definition{
		Type:            commandTypeUpdate,
		Owner:           command.OwnerCore,
		ValidatePayload: validateUpdatePayload,
	})
}

// RegisterEvents registers invite events with the shared registry.
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
		Type:            eventTypeClaimed,
		Owner:           event.OwnerCore,
		ValidatePayload: validateClaimPayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeRevoked,
		Owner:           event.OwnerCore,
		ValidatePayload: validateRevokePayload,
	}); err != nil {
		return err
	}
	return registry.Register(event.Definition{
		Type:            eventTypeUpdated,
		Owner:           event.OwnerCore,
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
