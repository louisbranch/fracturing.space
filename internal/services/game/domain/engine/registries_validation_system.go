package engine

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

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
// registered by a module is handled by the module's decider.
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
		key := mod.ID() + "@" + mod.Version()
		typer, ok := decider.(module.CommandTyper)
		if !ok {
			// If the module has registered system commands but its decider
			// does not implement CommandTyper, the coverage check cannot
			// verify handler completeness — fail loudly instead of silently
			// skipping.
			if len(systemCommands[key]) > 0 {
				return fmt.Errorf("module %s has %d registered system commands but its decider does not implement CommandTyper",
					key, len(systemCommands[key]))
			}
			continue
		}
		handled := make(map[command.Type]struct{})
		for _, t := range typer.DeciderHandledCommands() {
			handled[t] = struct{}{}
		}
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
// system adapter.
func ValidateAdapterEventCoverage(modules *module.Registry, adapters *bridge.AdapterRegistry, events *event.Registry) error {
	if modules == nil || adapters == nil || events == nil {
		return fmt.Errorf("module, adapter, and event registries are required")
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

// ValidateStateFactoryDeterminism verifies that each module's StateFactory
// produces identical output across repeated calls with the same input for
// both NewSnapshotState and NewCharacterState.
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

// ValidateSystemRouterDefinitionParity verifies that every type in a system
// module's Folder.FoldHandledTypes() and the corresponding adapter's
// HandledTypes() has a matching entry in the module's EmittableEventTypes()
// with the appropriate intent.
func ValidateSystemRouterDefinitionParity(
	modules *module.Registry,
	adapters *bridge.AdapterRegistry,
	events *event.Registry,
) error {
	if modules == nil || adapters == nil || events == nil {
		return fmt.Errorf("module, adapter, and event registries are required")
	}

	var orphaned []string

	for _, mod := range modules.List() {
		// Build set of emittable types with replay intent for fold parity.
		emittableReplay := make(map[event.Type]struct{})
		emittableProjection := make(map[event.Type]struct{})
		for _, t := range mod.EmittableEventTypes() {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Intent == event.IntentProjectionAndReplay || def.Intent == event.IntentReplayOnly {
				emittableReplay[t] = struct{}{}
			}
			if def.Intent == event.IntentProjectionAndReplay {
				emittableProjection[t] = struct{}{}
			}
		}

		// Check fold handlers against emittable replay set.
		if folder := mod.Folder(); folder != nil {
			for _, t := range folder.FoldHandledTypes() {
				if _, ok := emittableReplay[t]; !ok {
					orphaned = append(orphaned, string(t)+" (fold)")
				}
			}
		}
	}

	// Check adapter handlers against emittable projection set.
	for _, adapter := range adapters.Adapters() {
		mod := modules.Get(adapter.ID(), adapter.Version())
		if mod == nil {
			continue
		}
		emittableProjection := make(map[event.Type]struct{})
		for _, t := range mod.EmittableEventTypes() {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Intent == event.IntentProjectionAndReplay {
				emittableProjection[t] = struct{}{}
			}
		}
		for _, t := range adapter.HandledTypes() {
			if _, ok := emittableProjection[t]; !ok {
				orphaned = append(orphaned, string(t)+" (adapter)")
			}
		}
	}

	if len(orphaned) > 0 {
		return fmt.Errorf("system router handlers for types not in EmittableEventTypes: %s",
			strings.Join(orphaned, ", "))
	}
	return nil
}
