package scene

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

var sceneCommandContracts = []commandContract{
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
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeEnd,
			Owner:           command.OwnerCore,
			ValidatePayload: validateEndPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterAdd,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterAddedPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterRemove,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterRemovedPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterTransfer,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterTransferPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeTransition,
			Owner:           command.OwnerCore,
			ValidatePayload: validateTransitionPayload,
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateOpen,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateOpenedPayload,
			Gate: command.GatePolicy{
				Scope: command.GateScopeScene,
			},
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateResolve,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateResolvedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeGateAbandon,
			Owner:           command.OwnerCore,
			ValidatePayload: validateGateAbandonedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSpotlightSet,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSpotlightSetPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeSpotlightClear,
			Owner:           command.OwnerCore,
			ValidatePayload: validateSpotlightClearedPayload,
			Gate: command.GatePolicy{
				Scope:         command.GateScopeScene,
				AllowWhenOpen: true,
			},
		},
	},
}

var sceneEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeCreated,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCreatePayload,
			Intent:          event.IntentProjectionAndReplay,
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
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeEnded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateEndPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeCharacterAdded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCharacterAddedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeCharacterRemoved,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCharacterRemovedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGateOpened,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateOpenedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGateResolved,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateResolvedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeGateAbandoned,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateGateAbandonedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeSpotlightSet,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSpotlightSetPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeSpotlightCleared,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateSpotlightClearedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

// RegisterCommands registers scene commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range sceneCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEvents registers scene events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range sceneEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the scene decider can emit.
func EmittableEventTypes() []event.Type {
	return sceneEventTypes(func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the scene decider handles.
func DeciderHandledCommands() []command.Type {
	types := make([]command.Type, 0, len(sceneCommandContracts))
	for _, contract := range sceneCommandContracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

// ProjectionHandledTypes returns the scene event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return sceneEventTypes(func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

func sceneEventTypes(include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(sceneEventContracts))
	for _, contract := range sceneEventContracts {
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

func validateUpdatePayload(raw json.RawMessage) error {
	var payload UpdatePayload
	return json.Unmarshal(raw, &payload)
}

func validateEndPayload(raw json.RawMessage) error {
	var payload EndPayload
	return json.Unmarshal(raw, &payload)
}

func validateCharacterAddedPayload(raw json.RawMessage) error {
	var payload CharacterAddedPayload
	return json.Unmarshal(raw, &payload)
}

func validateCharacterRemovedPayload(raw json.RawMessage) error {
	var payload CharacterRemovedPayload
	return json.Unmarshal(raw, &payload)
}

func validateCharacterTransferPayload(raw json.RawMessage) error {
	var payload CharacterTransferPayload
	return json.Unmarshal(raw, &payload)
}

func validateTransitionPayload(raw json.RawMessage) error {
	var payload TransitionPayload
	return json.Unmarshal(raw, &payload)
}

func validateGateOpenedPayload(raw json.RawMessage) error {
	var payload GateOpenedPayload
	return json.Unmarshal(raw, &payload)
}

func validateGateResolvedPayload(raw json.RawMessage) error {
	var payload GateResolvedPayload
	return json.Unmarshal(raw, &payload)
}

func validateGateAbandonedPayload(raw json.RawMessage) error {
	var payload GateAbandonedPayload
	return json.Unmarshal(raw, &payload)
}

func validateSpotlightSetPayload(raw json.RawMessage) error {
	var payload SpotlightSetPayload
	return json.Unmarshal(raw, &payload)
}

func validateSpotlightClearedPayload(raw json.RawMessage) error {
	var payload SpotlightClearedPayload
	return json.Unmarshal(raw, &payload)
}
