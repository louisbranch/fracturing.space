package session

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sessionCommandContracts = appendSessionCommandContracts(
	sessionLifecycleCommandContracts,
	sessionGateCommandContracts,
	sessionSpotlightCommandContracts,
	sessionInteractionCommandContracts,
)

var sessionEventContracts = appendSessionEventContracts(
	sessionLifecycleEventContracts,
	sessionGateEventContracts,
	sessionSpotlightEventContracts,
	sessionInteractionEventContracts,
)

// RegisterCommands registers session commands with the shared registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	for _, contract := range sessionCommandContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}

// EmittableEventTypes returns all event types the session decider can emit.
func EmittableEventTypes() []event.Type {
	return sessionEventTypes(sessionEventContracts, func(contract eventProjectionContract) bool {
		return contract.emittable
	})
}

// DeciderHandledCommands returns all command types the session decider handles.
func DeciderHandledCommands() []command.Type {
	return sessionCommandTypes(sessionCommandContracts)
}

// ProjectionHandledTypes returns the session event types that require
// projection handlers (IntentProjectionAndReplay).
func ProjectionHandledTypes() []event.Type {
	return sessionEventTypes(sessionEventContracts, func(contract eventProjectionContract) bool {
		return contract.projection
	})
}

// RegisterEvents registers session events with the shared registry.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	for _, contract := range sessionEventContracts {
		if err := registry.Register(contract.definition); err != nil {
			return err
		}
	}
	return nil
}
