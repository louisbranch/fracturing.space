package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

// Registries bundles the command/event/system registries.
type Registries struct {
	Commands *command.Registry
	Events   *event.Registry
	Systems  *system.Registry
}

// BuildRegistries registers core and system modules.
//
// This is the shared bootstrap point where all command/event contracts become a
// single validated registry consumed by the write handler and projections.
func BuildRegistries(modules ...system.Module) (Registries, error) {
	commandRegistry := command.NewRegistry()
	eventRegistry := event.NewRegistry()
	systemRegistry := system.NewRegistry()

	if err := campaign.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := action.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := session.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := participant.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := invite.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := character.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}

	if err := campaign.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := action.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := session.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := participant.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := invite.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := character.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := eventRegistry.RegisterAlias(participant.EventTypeSeatReassignedLegacy, participant.EventTypeSeatReassigned); err != nil {
		return Registries{}, err
	}

	for _, module := range modules {
		if err := systemRegistry.Register(module); err != nil {
			return Registries{}, err
		}
		beforeCommands := commandTypeSet(commandRegistry.ListDefinitions())
		if err := module.RegisterCommands(commandRegistry); err != nil {
			return Registries{}, err
		}
		beforeEvents := eventTypeSet(eventRegistry.ListDefinitions())
		if err := module.RegisterEvents(eventRegistry); err != nil {
			return Registries{}, err
		}
		if err := validateModuleSystemTypePrefixes(module, beforeCommands, beforeEvents, commandRegistry.ListDefinitions(), eventRegistry.ListDefinitions()); err != nil {
			return Registries{}, err
		}
	}

	return Registries{
		Commands: commandRegistry,
		Events:   eventRegistry,
		Systems:  systemRegistry,
	}, nil
}

// validateModuleSystemTypePrefixes enforces system namespace naming for system-owned
// command/event types.
func validateModuleSystemTypePrefixes(
	module system.Module,
	knownCommands map[command.Type]struct{},
	knownEvents map[event.Type]struct{},
	commands []command.Definition,
	events []event.Definition,
) error {
	moduleID := strings.TrimSpace(module.ID())
	namespace := naming.NormalizeSystemNamespace(moduleID)
	if namespace == "" {
		return fmt.Errorf("system module id is required for naming validation")
	}
	expectedPrefix := "sys." + namespace + "."

	for _, definition := range commands {
		if definition.Owner != command.OwnerSystem {
			continue
		}
		if _, exists := knownCommands[definition.Type]; exists {
			continue
		}
		name := string(definition.Type)
		if strings.HasPrefix(name, expectedPrefix) {
			continue
		}
		return fmt.Errorf("system module %s command %s must use %s prefix", moduleID, definition.Type, expectedPrefix)
	}

	for _, definition := range events {
		if definition.Owner != event.OwnerSystem {
			continue
		}
		if _, exists := knownEvents[definition.Type]; exists {
			continue
		}
		name := string(definition.Type)
		if strings.HasPrefix(name, expectedPrefix) {
			continue
		}
		return fmt.Errorf("system module %s event %s must use %s prefix", moduleID, definition.Type, expectedPrefix)
	}
	return nil
}

// commandTypeSet creates a set view for prefix validation comparisons.
func commandTypeSet(definitions []command.Definition) map[command.Type]struct{} {
	result := make(map[command.Type]struct{}, len(definitions))
	for _, definition := range definitions {
		result[definition.Type] = struct{}{}
	}
	return result
}

// eventTypeSet creates a set view for prefix validation comparisons.
func eventTypeSet(definitions []event.Definition) map[event.Type]struct{} {
	result := make(map[event.Type]struct{}, len(definitions))
	for _, definition := range definitions {
		result[definition.Type] = struct{}{}
	}
	return result
}
