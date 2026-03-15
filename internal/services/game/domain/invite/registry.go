package invite

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

var inviteCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeCreate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCreatePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
			Target:          command.TargetEntity("invite", "invite_id"),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeClaim,
			Owner:           command.OwnerCore,
			ValidatePayload: validateClaimPayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
			Target:          command.TargetEntity("invite", "invite_id"),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeDecline,
			Owner:           command.OwnerCore,
			ValidatePayload: validateDeclinePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
			Target:          command.TargetEntity("invite", "invite_id"),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeRevoke,
			Owner:           command.OwnerCore,
			ValidatePayload: validateRevokePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
			Target:          command.TargetEntity("invite", "invite_id"),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeUpdate,
			Owner:           command.OwnerCore,
			ValidatePayload: validateUpdatePayload,
			ActiveSession:   command.BlockedDuringActiveSession(),
			Target:          command.TargetEntity("invite", "invite_id"),
		},
	},
}

var inviteEventContracts = []eventProjectionContract{
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
			Type:            EventTypeClaimed,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateClaimPayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeDeclined,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateDeclinePayload,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeRevoked,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateRevokePayload,
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
}

// RegisterCommands registers invite commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range inviteCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the invite decider can emit.
func EmittableEventTypes() []event.Type {
	return inviteEventTypes(func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the invite decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(inviteCommandContracts))
	for _, contract := range inviteCommandContracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the invite event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return inviteEventTypes(func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

// RegisterEvents registers invite events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range inviteEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

func inviteEventTypes(include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(inviteEventContracts))
	for _, contract := range inviteEventContracts {
		if include(contract) {
			types = append(types, contract.definition.Type)
		}
	}
	return types
}

func validateCreatePayload(raw json.RawMessage) error {
	var payload CreatePayload
	return json.Unmarshal(raw, &payload)
}

func validateClaimPayload(raw json.RawMessage) error {
	var payload ClaimPayload
	return json.Unmarshal(raw, &payload)
}

func validateDeclinePayload(raw json.RawMessage) error {
	var payload DeclinePayload
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
