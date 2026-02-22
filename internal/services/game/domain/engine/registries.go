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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// Registries bundles the command/event/system registries.
type Registries struct {
	Commands *command.Registry
	Events   *event.Registry
	Systems  *module.Registry
}

// BuildRegistries registers core and system modules.
//
// This is the shared bootstrap point where all command/event contracts become a
// single validated registry consumed by the write handler and projections.
func BuildRegistries(modules ...module.Module) (Registries, error) {
	commandRegistry := command.NewRegistry()
	eventRegistry := event.NewRegistry()
	systemRegistry := module.NewRegistry()

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

	if err := ValidateFoldCoverage(eventRegistry); err != nil {
		return Registries{}, err
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

// validateCoreEmittableEventTypes ensures every event type a core domain
// decider declares as emittable is registered in the event registry.
func validateCoreEmittableEventTypes(events *event.Registry) error {
	var missing []string
	for _, types := range [][]event.Type{
		campaign.EmittableEventTypes(),
		session.EmittableEventTypes(),
		action.EmittableEventTypes(),
		character.EmittableEventTypes(),
		participant.EmittableEventTypes(),
		invite.EmittableEventTypes(),
	} {
		for _, t := range types {
			if _, ok := events.Definition(t); !ok {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core emittable event types not in registry: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

// validateEmittableEventTypes ensures every event type a module declares as
// emittable is registered in the event registry. This catches missing
// RegisterEvents calls at startup instead of at runtime when a code path fires.
func validateEmittableEventTypes(mod module.Module, events *event.Registry) error {
	emittable := mod.EmittableEventTypes()
	if len(emittable) == 0 {
		return nil
	}
	var missing []string
	for _, t := range emittable {
		if _, ok := events.Definition(t); !ok {
			missing = append(missing, string(t))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("system module %s declares emittable event types not in registry: %s",
			mod.ID(), strings.Join(missing, ", "))
	}
	return nil
}

// ValidateFoldCoverage verifies that every core event with IntentProjectionAndReplay
// or IntentReplayOnly has a fold handler declared via FoldHandledTypes in the domain
// packages.
//
// This is a startup-time safety check: if a developer adds a new event that affects
// aggregate state and forgets the fold case, the server refuses to start.
func ValidateFoldCoverage(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for fold coverage validation")
	}

	handled := make(map[event.Type]struct{})
	for _, types := range [][]event.Type{
		campaign.FoldHandledTypes(),
		session.FoldHandledTypes(),
		action.FoldHandledTypes(),
		character.FoldHandledTypes(),
		participant.FoldHandledTypes(),
		invite.FoldHandledTypes(),
	} {
		for _, t := range types {
			handled[t] = struct{}{}
		}
	}

	var missing []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerCore {
			continue
		}
		if def.Intent != event.IntentProjectionAndReplay && def.Intent != event.IntentReplayOnly {
			continue
		}
		if _, ok := handled[def.Type]; !ok {
			missing = append(missing, string(def.Type))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core replay events missing fold handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateProjectionCoverage verifies that every core IntentProjectionAndReplay
// event has a projection handler declared via ProjectionHandledTypes.
//
// This is a startup-time safety check: if a developer adds a new event with
// IntentProjectionAndReplay and forgets the projection switch case, the server
// refuses to start.
func ValidateProjectionCoverage(events *event.Registry, handledTypes []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for projection coverage validation")
	}

	handled := make(map[event.Type]struct{})
	for _, t := range handledTypes {
		handled[t] = struct{}{}
	}

	var missing []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerCore {
			continue
		}
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		// A type is covered if it is handled directly or if the registry
		// resolves it (via alias) to a handled canonical type.
		if _, ok := handled[def.Type]; ok {
			continue
		}
		if resolved := events.Resolve(def.Type); resolved != def.Type {
			if _, ok := handled[resolved]; ok {
				continue
			}
		}
		missing = append(missing, string(def.Type))
	}
	if len(missing) > 0 {
		return fmt.Errorf("core projection-and-replay events missing projection handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateAdapterEventCoverage verifies that every system module's emittable
// event types with IntentProjectionAndReplay are handled by the corresponding
// system adapter. This catches the case where a module declares/registers/emits
// events but no adapter handler exists, causing runtime errors.
func ValidateAdapterEventCoverage(modules *module.Registry, adapters *systems.AdapterRegistry, events *event.Registry) error {
	if modules == nil || adapters == nil || events == nil {
		return fmt.Errorf("module registry, adapter registry, and event registry are required")
	}

	// Build a set of types each adapter handles.
	adapterHandled := make(map[event.Type]struct{})
	for _, adapter := range adapters.Adapters() {
		for _, t := range adapter.HandledTypes() {
			adapterHandled[t] = struct{}{}
		}
	}

	var missing []string
	for _, mod := range modules.List() {
		for _, t := range mod.EmittableEventTypes() {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Intent != event.IntentProjectionAndReplay {
				continue
			}
			if _, ok := adapterHandled[t]; !ok {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("system emittable events missing adapter handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}
