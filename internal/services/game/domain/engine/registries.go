package engine

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
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

	if err := ValidateSystemFoldCoverage(systemRegistry, eventRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateDeciderCommandCoverage(systemRegistry, commandRegistry); err != nil {
		return Registries{}, err
	}

	if err := ValidateFoldCoverage(eventRegistry); err != nil {
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

	if missing := eventRegistry.MissingPayloadValidators(); len(missing) > 0 {
		names := make([]string, len(missing))
		for i, t := range missing {
			names[i] = string(t)
		}
		log.Printf("WARNING: non-audit events without payload validators: %s", strings.Join(names, ", "))
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
	for _, domain := range CoreDomains() {
		for _, t := range domain.EmittableEventTypes() {
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
	for _, domain := range CoreDomains() {
		for _, t := range domain.FoldHandledTypes() {
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

// ValidateSystemFoldCoverage verifies that every system module's emittable
// event types with IntentProjectionAndReplay or IntentReplayOnly are handled
// by the module's folder. This is the system-event counterpart of
// ValidateFoldCoverage, which covers core domains.
func ValidateSystemFoldCoverage(modules *module.Registry, events *event.Registry) error {
	if modules == nil || events == nil {
		return fmt.Errorf("module registry and event registry are required")
	}

	var missing []string
	for _, mod := range modules.List() {
		folder := mod.Folder()
		if folder == nil {
			continue
		}
		handled := make(map[event.Type]struct{})
		for _, t := range folder.FoldHandledTypes() {
			handled[t] = struct{}{}
		}
		for _, t := range mod.EmittableEventTypes() {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Intent != event.IntentProjectionAndReplay && def.Intent != event.IntentReplayOnly {
				continue
			}
			if _, ok := handled[t]; !ok {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("system emittable events missing folder fold handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateDeciderCommandCoverage verifies that every system command type
// registered by a module is handled by the module's decider. Only modules
// whose decider implements module.CommandTyper are checked; modules that
// don't declare handled commands are silently skipped.
//
// This closes the gap where a developer registers a command but forgets
// the decider switch case — the server refuses to start instead of
// returning a generic runtime rejection.
func ValidateDeciderCommandCoverage(modules *module.Registry, commands *command.Registry) error {
	if modules == nil || commands == nil {
		return fmt.Errorf("module registry and command registry are required")
	}

	// Build a set of system-owned command types each module registered.
	systemCommands := make(map[string]map[command.Type]struct{})
	for _, def := range commands.ListDefinitions() {
		if def.Owner != command.OwnerSystem {
			continue
		}
		for _, mod := range modules.List() {
			namespace := "sys." + naming.NormalizeSystemNamespace(mod.ID()) + "."
			if strings.HasPrefix(string(def.Type), namespace) {
				key := mod.ID() + "@" + mod.Version()
				if systemCommands[key] == nil {
					systemCommands[key] = make(map[command.Type]struct{})
				}
				systemCommands[key][def.Type] = struct{}{}
			}
		}
	}

	var missing []string
	for _, mod := range modules.List() {
		decider := mod.Decider()
		if decider == nil {
			continue
		}
		typer, ok := decider.(module.CommandTyper)
		if !ok {
			continue
		}
		handled := make(map[command.Type]struct{})
		for _, t := range typer.DeciderHandledCommands() {
			handled[t] = struct{}{}
		}
		key := mod.ID() + "@" + mod.Version()
		for ct := range systemCommands[key] {
			if _, ok := handled[ct]; !ok {
				missing = append(missing, string(ct))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("system commands missing decider handlers: %s", strings.Join(missing, ", "))
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

// ValidateNoFoldHandlersForAuditOnlyEvents verifies that no fold handler
// exists for an event with IntentAuditOnly. Such a handler would be dead
// code — the aggregate folder skips audit-only events at runtime, so a
// handler would never execute.
func ValidateNoFoldHandlersForAuditOnlyEvents(events *event.Registry, foldHandled []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for audit-only fold guard")
	}

	var dead []string
	for _, t := range foldHandled {
		def, ok := events.Definition(t)
		if !ok {
			continue
		}
		if def.Intent == event.IntentAuditOnly {
			dead = append(dead, string(t))
		}
	}
	if len(dead) > 0 {
		return fmt.Errorf("fold handlers registered for audit-only events (dead code): %s",
			strings.Join(dead, ", "))
	}
	return nil
}

// ValidateNoProjectionHandlersForNonProjectionEvents verifies that no
// projection handler exists for an event with IntentAuditOnly or
// IntentReplayOnly. Such handlers would be dead code — the projection
// applier skips non-projection events at runtime.
func ValidateNoProjectionHandlersForNonProjectionEvents(events *event.Registry, projectionHandled []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for projection intent guard")
	}

	var dead []string
	for _, t := range projectionHandled {
		def, ok := events.Definition(t)
		if !ok {
			continue
		}
		if def.Intent == event.IntentAuditOnly || def.Intent == event.IntentReplayOnly {
			dead = append(dead, string(t))
		}
	}
	if len(dead) > 0 {
		return fmt.Errorf("projection handlers registered for non-projection events (dead code): %s",
			strings.Join(dead, ", "))
	}
	return nil
}

// entityKeyedDomains returns the core domains whose fold types require entity
// addressing (EntityID != "" guard in the aggregate folder). Adding a new
// entity-keyed domain here ensures ValidateEntityKeyedAddressing catches
// missing AddressingPolicyEntityTarget at startup.
func entityKeyedDomains() []CoreDomain {
	var keyed []CoreDomain
	for _, d := range CoreDomains() {
		switch d.Name() {
		case "participant", "character", "invite":
			keyed = append(keyed, d)
		}
	}
	return keyed
}

// ValidateEntityKeyedAddressing verifies that every entity-keyed fold type
// (participant, character, invite) has AddressingPolicyEntityTarget in its
// event definition. This catches the case where a developer registers an
// entity-keyed event but forgets to set addressing policy, causing the
// aggregate folder to silently skip the fold when EntityID is empty.
func ValidateEntityKeyedAddressing(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for entity-keyed addressing validation")
	}

	var missing []string
	for _, domain := range entityKeyedDomains() {
		for _, t := range domain.FoldHandledTypes() {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Addressing != event.AddressingPolicyEntityTarget {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("entity-keyed fold types missing AddressingPolicyEntityTarget: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

// ValidateStateFactoryDeterminism verifies that each module's StateFactory
// produces identical output across repeated calls with the same input for
// both NewSnapshotState and NewCharacterState. A non-deterministic factory
// (e.g. one using time.Now() or random IDs) would cause silent replay
// divergence.
func ValidateStateFactoryDeterminism(modules *module.Registry) error {
	if modules == nil {
		return fmt.Errorf("module registry is required for state factory determinism check")
	}

	const checkID = "determinism-check"
	for _, mod := range modules.List() {
		factory := mod.StateFactory()
		if factory == nil {
			continue
		}
		label := mod.ID() + "@" + mod.Version()

		firstSnap, err := factory.NewSnapshotState(checkID)
		if err != nil {
			return fmt.Errorf("state factory %s NewSnapshotState error: %w", label, err)
		}
		secondSnap, err := factory.NewSnapshotState(checkID)
		if err != nil {
			return fmt.Errorf("state factory %s NewSnapshotState error: %w", label, err)
		}
		if !reflect.DeepEqual(firstSnap, secondSnap) {
			return fmt.Errorf("state factory determinism check failed for %s: NewSnapshotState returned different results", label)
		}

		firstChar, err := factory.NewCharacterState(checkID, checkID, checkID)
		if err != nil {
			return fmt.Errorf("state factory %s NewCharacterState error: %w", label, err)
		}
		secondChar, err := factory.NewCharacterState(checkID, checkID, checkID)
		if err != nil {
			return fmt.Errorf("state factory %s NewCharacterState error: %w", label, err)
		}
		if !reflect.DeepEqual(firstChar, secondChar) {
			return fmt.Errorf("state factory determinism check failed for %s: NewCharacterState returned different results", label)
		}
	}
	return nil
}

// ValidateAggregateFoldDispatch verifies that every core event type declared
// in CoreDomains().FoldHandledTypes is actually wired into the aggregate
// applier's fold dispatch sets. This catches the case where a developer adds
// FoldHandledTypes for a new domain but forgets to wire initFoldSets and the
// if-block in Apply.
func ValidateAggregateFoldDispatch(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for aggregate fold dispatch validation")
	}

	applier := &aggregate.Folder{}
	dispatched := make(map[event.Type]struct{})
	for _, t := range applier.FoldDispatchedTypes() {
		dispatched[t] = struct{}{}
	}

	declared := make(map[event.Type]struct{})
	for _, domain := range CoreDomains() {
		for _, t := range domain.FoldHandledTypes() {
			declared[t] = struct{}{}
		}
	}

	var missing []string
	for t := range declared {
		if _, ok := dispatched[t]; !ok {
			missing = append(missing, string(t))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core fold types declared but not dispatched by aggregate applier: %s",
			strings.Join(missing, ", "))
	}
	return nil
}
