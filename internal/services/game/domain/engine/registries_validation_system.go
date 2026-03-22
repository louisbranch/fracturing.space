package engine

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
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
		sort.Strings(missing)
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
		sort.Strings(missing)
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

	moduleList := modules.List()
	systemCommands := collectSystemCommandsByModule(moduleList, commands.ListDefinitions())

	var missing []string
	for _, mod := range moduleList {
		decider := mod.Decider()
		if decider == nil {
			continue
		}
		key := moduleVersionKey(mod)
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
		handled := commandTypeSetFromSlice(typer.DeciderHandledCommands())
		for ct := range systemCommands[key] {
			if _, ok := handled[ct]; !ok {
				missing = append(missing, string(ct))
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
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
		sort.Strings(missing)
		return fmt.Errorf("system emittable events missing adapter handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateSystemMetadataConsistency verifies that every OwnerSystem event
// has a corresponding registered system module whose namespace matches the
// event type prefix. This is defense-in-depth for the system routing fix
// (A5): if a system event slips through without a matching module, it would
// hit the "no system module registered" error at fold time.
func ValidateSystemMetadataConsistency(events *event.Registry, modules *module.Registry) error {
	if events == nil || modules == nil {
		return fmt.Errorf("event and module registries are required")
	}

	var orphaned []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerSystem {
			continue
		}
		// Extract namespace from the event type (sys.<namespace>.<rest>).
		typeName := string(def.Type)
		namespace, ok := naming.NamespaceFromType(typeName)
		if !ok || namespace == "" {
			orphaned = append(orphaned, typeName+" (no sys. prefix)")
			continue
		}
		// Check if any registered module matches this namespace.
		found := false
		for _, mod := range modules.List() {
			if naming.NormalizeSystemNamespace(mod.ID()) == namespace {
				found = true
				break
			}
		}
		if !found {
			orphaned = append(orphaned, typeName)
		}
	}
	if len(orphaned) > 0 {
		sort.Strings(orphaned)
		return fmt.Errorf("system event types without matching module: %s",
			strings.Join(orphaned, ", "))
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

// ValidateStateFactoryFoldCompatibility verifies that each module's
// StateFactory produces state whose type is accepted by the module's Folder.
// Because StateFactory returns `any` and Folder.Fold takes `any`, a module
// author can wire a FooState factory with a FoldRouter[*BarState] and only
// discover the mismatch at runtime when the first event folds. This validator
// catches that class of error at startup by feeding factory output into the
// fold and distinguishing a type-assertion error (incompatible) from an
// unhandled-event-type error (expected, since the probe event is synthetic).
func ValidateStateFactoryFoldCompatibility(modules *module.Registry) error {
	if modules == nil {
		return fmt.Errorf("module registry is required for state factory fold compatibility check")
	}

	const testCampaignID = "fold-compat-check"
	probeEvent := event.Event{Type: "nonexistent-validation-check"}

	for _, mod := range modules.List() {
		factory := mod.StateFactory()
		if factory == nil {
			continue
		}
		folder := mod.Folder()
		if folder == nil {
			continue
		}
		label := mod.ID() + "@" + mod.Version()

		state, err := factory.NewSnapshotState(testCampaignID)
		if err != nil {
			return fmt.Errorf("state factory %s NewSnapshotState error: %w", label, err)
		}

		_, foldErr := folder.Fold(state, probeEvent)
		if foldErr == nil {
			// An unknown event type should always error — a nil error is
			// unexpected but not a compatibility failure.
			continue
		}

		// The fold router returns "unhandled fold event type" when the type
		// assertion succeeded but no handler matched the synthetic event
		// type. Any other error indicates the state type is incompatible.
		if !strings.Contains(foldErr.Error(), "unhandled fold event type") {
			return fmt.Errorf("state factory / fold type mismatch for %s: factory produces %T but folder rejects it: %v",
				label, state, foldErr)
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
		sort.Strings(orphaned)
		return fmt.Errorf("system router handlers for types not in EmittableEventTypes: %s",
			strings.Join(orphaned, ", "))
	}
	return nil
}

// collectSystemCommandsByModule maps each module (`id@version`) to the
// system-owned command types currently registered for its namespace.
func collectSystemCommandsByModule(
	modules []module.Module,
	definitions []command.Definition,
) map[string]map[command.Type]struct{} {
	modulePrefixes := make(map[string]string, len(modules))
	for _, mod := range modules {
		modulePrefixes[moduleVersionKey(mod)] = "sys." + naming.NormalizeSystemNamespace(mod.ID()) + "."
	}

	coverage := make(map[string]map[command.Type]struct{}, len(modules))
	for _, definition := range definitions {
		if definition.Owner != command.OwnerSystem {
			continue
		}
		typeName := string(definition.Type)
		for key, prefix := range modulePrefixes {
			if !strings.HasPrefix(typeName, prefix) {
				continue
			}
			if coverage[key] == nil {
				coverage[key] = make(map[command.Type]struct{})
			}
			coverage[key][definition.Type] = struct{}{}
		}
	}
	return coverage
}

func commandTypeSetFromSlice(types []command.Type) map[command.Type]struct{} {
	set := make(map[command.Type]struct{}, len(types))
	for _, commandType := range types {
		set[commandType] = struct{}{}
	}
	return set
}

func moduleVersionKey(mod module.Module) string {
	return mod.ID() + "@" + mod.Version()
}
