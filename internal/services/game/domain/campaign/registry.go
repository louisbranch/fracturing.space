package campaign

import (
	"encoding/json"
	"errors"
	"strings"

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

var campaignCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeCreate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCreatePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCreateWithParticipants,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCreateWithParticipantsPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeUpdate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateUpdatePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAIBind,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAIBindPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAIUnbind,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAIUnbindPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeAIAuthRotate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateAIAuthRotatePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeFork,
			Owner:           command.OwnerCore,
			ValidatePayload: validateForkPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeEnd,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEmptyPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeArchive,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEmptyPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeRestore,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEmptyPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
		},
	},
}

var campaignEventContracts = []eventProjectionContract{
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
			Type:            EventTypeAIBound,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAIBindPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAIUnbound,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAIUnbindPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeAIAuthRotated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateAIAuthRotatePayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeForked,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateForkPayload,
		},
		emittable:  true,
		projection: true,
	},
}

// RegisterCommands registers campaign commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range campaignCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the campaign decider can emit.
func EmittableEventTypes() []event.Type {
	return campaignEventTypes(func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the campaign decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(campaignCommandContracts))
	for _, contract := range campaignCommandContracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the campaign event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return campaignEventTypes(func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

// RegisterEvents registers campaign events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range campaignEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

func campaignEventTypes(include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(campaignEventContracts))
	for _, contract := range campaignEventContracts {
		if include(contract) {
			types = append(types, contract.definition.Type)
		}
	}
	return types
}

// validateCreatePayload ensures command payloads match the campaign create shape.
func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateCreateWithParticipantsPayload ensures workflow payload includes
// campaign create input and participant bootstrap inputs.
func validateCreateWithParticipantsPayload(raw json.RawMessage) error {
	var payload CreateWithParticipantsPayload
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

// validateAIBindPayload ensures bind payloads include ai_agent_id.
func validateAIBindPayload(raw json.RawMessage) error {
	var payload AIBindPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AIAgentID) == "" {
		return errors.New("ai_agent_id is required")
	}
	return nil
}

// validateAIUnbindPayload ensures unbind payload shape is accepted.
func validateAIUnbindPayload(raw json.RawMessage) error {
	var payload AIUnbindPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return nil
}

// validateAIAuthRotatePayload ensures rotate payloads include a reason string.
func validateAIAuthRotatePayload(raw json.RawMessage) error {
	var payload AIAuthRotatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

// validateForkPayload ensures fork payloads include required identifiers.
func validateForkPayload(raw json.RawMessage) error {
	var payload ForkPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ParentCampaignID.String()) == "" {
		return errors.New("parent_campaign_id is required")
	}
	if strings.TrimSpace(payload.OriginCampaignID.String()) == "" {
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
