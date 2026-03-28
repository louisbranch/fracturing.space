package engine

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type moduleRegistrationBaseline struct {
	knownCommands map[command.Type]struct{}
	knownEvents   map[event.Type]struct{}
}

type systemModuleRegistrationStep func(registryBootstrap, module.Module, moduleRegistrationBaseline) error

type namedSystemModuleRegistrationStep struct {
	name string
	run  systemModuleRegistrationStep
}

// registrySystemModuleRegistrar owns the module-scoped registration phase and
// its module-local contract checks.
type registrySystemModuleRegistrar struct{}

// Register loads each system module into the shared registries and validates
// the command/event types newly introduced by that module.
func (registrySystemModuleRegistrar) Register(bootstrap registryBootstrap) error {
	return bootstrap.registerSystemModules()
}

// registerSystemModules executes the module registration phase and validates
// new system-owned command/event types against namespace and emit declarations.
func (b registryBootstrap) registerSystemModules() error {
	for _, mod := range b.modules {
		if err := b.registerSystemModule(mod); err != nil {
			return err
		}
	}
	return nil
}

// registerSystemModule runs all registration and validation steps for one
// module with explicit step names to improve startup error diagnostics.
func (b registryBootstrap) registerSystemModule(mod module.Module) error {
	if err := b.systemRegistry.Register(mod); err != nil {
		return err
	}

	baseline := captureModuleRegistrationBaseline(b.commandRegistry, b.eventRegistry)
	return runNamedSystemModuleRegistrationPipeline(
		b,
		mod,
		baseline,
		namedSystemModuleRegistrationStep{
			name: "register commands",
			run: func(state registryBootstrap, mod module.Module, _ moduleRegistrationBaseline) error {
				return mod.RegisterCommands(state.commandRegistry)
			},
		},
		namedSystemModuleRegistrationStep{
			name: "register events",
			run: func(state registryBootstrap, mod module.Module, _ moduleRegistrationBaseline) error {
				return mod.RegisterEvents(state.eventRegistry)
			},
		},
		namedSystemModuleRegistrationStep{
			name: "validate type namespace",
			run: func(state registryBootstrap, mod module.Module, baseline moduleRegistrationBaseline) error {
				return validateModuleSystemTypePrefixes(
					mod,
					baseline.knownCommands,
					baseline.knownEvents,
					state.commandRegistry.ListDefinitions(),
					state.eventRegistry.ListDefinitions(),
				)
			},
		},
		namedSystemModuleRegistrationStep{
			name: "validate emittable events",
			run: func(state registryBootstrap, mod module.Module, _ moduleRegistrationBaseline) error {
				return validateEmittableEventTypes(mod, state.eventRegistry)
			},
		},
	)
}

// captureModuleRegistrationBaseline snapshots command/event registries before
// a system module mutates them, so validation can compare only new definitions.
func captureModuleRegistrationBaseline(commands *command.Registry, events *event.Registry) moduleRegistrationBaseline {
	return moduleRegistrationBaseline{
		knownCommands: commandTypeSet(commands.ListDefinitions()),
		knownEvents:   eventTypeSet(events.ListDefinitions()),
	}
}

// runNamedSystemModuleRegistrationPipeline executes module registration steps
// in order and wraps failures with module and stage context.
func runNamedSystemModuleRegistrationPipeline(
	bootstrap registryBootstrap,
	mod module.Module,
	baseline moduleRegistrationBaseline,
	steps ...namedSystemModuleRegistrationStep,
) error {
	moduleName := moduleVersionLabel(mod)
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		if err := step.run(bootstrap, mod, baseline); err != nil {
			return fmt.Errorf("system module %s %s: %w", moduleName, step.name, err)
		}
	}
	return nil
}

// moduleVersionLabel formats `<id>@<version>` for startup diagnostics.
func moduleVersionLabel(mod module.Module) string {
	if moduleIsNil(mod) {
		return "<nil>"
	}
	id := strings.TrimSpace(mod.ID())
	version := strings.TrimSpace(mod.Version())
	if id == "" {
		id = "<unknown>"
	}
	if version == "" {
		return id
	}
	return id + "@" + version
}

// moduleIsNil reports whether mod is nil, including typed nils. reflect is
// required because a typed nil (e.g. (*MyModule)(nil)) passes the mod != nil
// check. reflect.ValueOf detects the underlying nil pointer.
func moduleIsNil(mod module.Module) bool {
	if mod == nil {
		return true
	}
	value := reflect.ValueOf(mod)
	return value.Kind() == reflect.Ptr && value.IsNil()
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

// validateModuleSystemTypePrefixes enforces system namespace naming for
// system-owned command/event types.
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
