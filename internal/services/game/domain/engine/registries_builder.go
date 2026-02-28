package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// BuildRegistries registers core and system modules.
//
// This is the shared bootstrap point where all command/event contracts become a
// single validated registry consumed by the write handler and projections.
func BuildRegistries(modules ...module.Module) (Registries, error) {
	commandRegistry := command.NewRegistry()
	eventRegistry := event.NewRegistry()
	systemRegistry := module.NewRegistry()

	for _, domain := range CoreDomains() {
		if err := domain.RegisterCommands(commandRegistry); err != nil {
			return Registries{}, fmt.Errorf("register %s commands: %w", domain.Name(), err)
		}
		if err := domain.RegisterEvents(eventRegistry); err != nil {
			return Registries{}, fmt.Errorf("register %s events: %w", domain.Name(), err)
		}
	}

	if err := eventRegistry.RegisterAlias(participant.EventTypeSeatReassignedLegacy, participant.EventTypeSeatReassigned); err != nil {
		return Registries{}, err
	}

	if err := validateCoreEmittableEventTypes(eventRegistry); err != nil {
		return Registries{}, err
	}

	for _, mod := range modules {
		if err := systemRegistry.Register(mod); err != nil {
			return Registries{}, err
		}
		beforeCommands := commandTypeSet(commandRegistry.ListDefinitions())
		if err := mod.RegisterCommands(commandRegistry); err != nil {
			return Registries{}, err
		}
		beforeEvents := eventTypeSet(eventRegistry.ListDefinitions())
		if err := mod.RegisterEvents(eventRegistry); err != nil {
			return Registries{}, err
		}
		if err := validateModuleSystemTypePrefixes(mod, beforeCommands, beforeEvents, commandRegistry.ListDefinitions(), eventRegistry.ListDefinitions()); err != nil {
			return Registries{}, err
		}
		if err := validateEmittableEventTypes(mod, eventRegistry); err != nil {
			return Registries{}, err
		}
	}

	if err := ValidateSystemReadinessCheckerCoverage(systemRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateSystemFoldCoverage(systemRegistry, eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateDeciderCommandCoverage(systemRegistry, commandRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateCoreDeciderCommandCoverage(commandRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateFoldCoverage(eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateAliasFoldCoverage(eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateAggregateFoldDispatch(eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateEntityKeyedAddressing(eventRegistry); err != nil {
		return Registries{}, err
	}

	// Collect all fold handled types (core + system) for intent-guard validation.
	var allFoldHandled []event.Type
	for _, domain := range CoreDomains() {
		allFoldHandled = append(allFoldHandled, domain.FoldHandledTypes()...)
	}
	for _, mod := range modules {
		if folder := mod.Folder(); folder != nil {
			allFoldHandled = append(allFoldHandled, folder.FoldHandledTypes()...)
		}
	}
	if err := ValidateNoFoldHandlersForAuditOnlyEvents(eventRegistry, allFoldHandled); err != nil {
		return Registries{}, err
	}

	if err := ValidateStateFactoryDeterminism(systemRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateSystemMetadataConsistency(eventRegistry, systemRegistry); err != nil {
		return Registries{}, err
	}

	projectionHandled := make([]event.Type, 0)
	for _, domain := range CoreDomains() {
		if domain.ProjectionHandledTypes != nil {
			projectionHandled = append(projectionHandled, domain.ProjectionHandledTypes()...)
		}
	}
	if err := ValidateProjectionRegistries(
		eventRegistry,
		systemRegistry,
		nil,
		projectionHandled,
	); err != nil {
		return Registries{}, err
	}

	missing := eventRegistry.MissingPayloadValidators()
	if len(missing) > 0 {
		names := make([]string, len(missing))
		for i, t := range missing {
			names[i] = string(t)
		}
		return Registries{}, fmt.Errorf("non-audit events without payload validators: %s", strings.Join(names, ", "))
	}

	return Registries{
		Commands: commandRegistry,
		Events:   eventRegistry,
		Systems:  systemRegistry,
	}, nil
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

// validateModuleSystemTypePrefixes enforces system namespace naming for system-owned
// command/event types.
func validateModuleSystemTypePrefixes(
	mod module.Module,
	knownCommands map[command.Type]struct{},
	knownEvents map[event.Type]struct{},
	commands []command.Definition,
	events []event.Definition,
) error {
	moduleID := strings.TrimSpace(mod.ID())
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
