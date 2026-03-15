package scene

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneCommandContracts = appendSceneCommandContracts(
	sceneLifecycleCommandContracts,
	sceneCharacterCommandContracts,
	sceneTransitionCommandContracts,
	sceneGateCommandContracts,
	sceneSpotlightCommandContracts,
	sceneInteractionCommandContracts,
)

var sceneEventContracts = appendSceneEventContracts(
	sceneLifecycleEventContracts,
	sceneCharacterEventContracts,
	sceneGateEventContracts,
	sceneSpotlightEventContracts,
	sceneInteractionEventContracts,
)

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
	return sceneEventTypes(sceneEventContracts, func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the scene decider handles.
func DeciderHandledCommands() []command.Type {
	return sceneCommandTypes(sceneCommandContracts)
}

// ProjectionHandledTypes returns the scene event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return sceneEventTypes(sceneEventContracts, func(contract eventProjectionContract) bool {
		return contract.projection
	})
}
