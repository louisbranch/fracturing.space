package campaign

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// RegisterCommands registers campaign commands with the shared registry.
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
		Type:            commandTypeFork,
		Owner:           command.OwnerCore,
		ValidatePayload: validateForkPayload,
	}); err != nil {
		return err
	}
	statusCommands := []command.Type{
		commandTypeEnd,
		commandTypeArchive,
		commandTypeRestore,
	}
	for _, cmdType := range statusCommands {
		if err := registry.Register(command.Definition{
			Type:            cmdType,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEmptyPayload,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEvents registers campaign events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeCreated,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateCreatePayload,
	}); err != nil {
		return err
	}
	if err := registry.Register(event.Definition{
		Type:            eventTypeForked,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateForkPayload,
	}); err != nil {
		return err
	}
	return registry.Register(event.Definition{
		Type:            eventTypeUpdated,
		Owner:           event.OwnerCore,
		Addressing:      event.AddressingPolicyEntityTarget,
		ValidatePayload: validateUpdatePayload,
	})
}

// validateCreatePayload ensures command payloads match the campaign create shape.
func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateUpdatePayload ensures update payloads match the campaign update shape.
func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateForkPayload ensures fork payloads include required identifiers.
func validateForkPayload(raw json.RawMessage) error {
	var payload ForkPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ParentCampaignID) == "" {
		return errors.New("parent_campaign_id is required")
	}
	if strings.TrimSpace(payload.OriginCampaignID) == "" {
		return errors.New("origin_campaign_id is required")
	}
	return nil
}

// validateEmptyPayload enforces payload-free lifecycle commands.
func validateEmptyPayload(raw json.RawMessage) error {
	if string(raw) != "{}" {
		return errors.New("payload must be empty")
	}
	return nil
}
