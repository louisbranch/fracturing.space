package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/naming"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type registryBootstrap struct {
	commandRegistry *command.Registry
	eventRegistry   *event.Registry
	systemRegistry  *module.Registry
	modules         []module.Module
}

// newRegistryBootstrap centralizes write-path registry allocation so tests can
// exercise the same bootstrap state container as production.
func newRegistryBootstrap(modules []module.Module) registryBootstrap {
	return registryBootstrap{
		commandRegistry: command.NewRegistry(),
		eventRegistry:   event.NewRegistry(),
		systemRegistry:  module.NewRegistry(),
		modules:         modules,
	}
}

// BuildRegistries registers core and system modules.
//
// This is the shared bootstrap point where all command/event contracts become a
// single validated registry consumed by the write handler and projections.
func BuildRegistries(modules ...module.Module) (Registries, error) {
	return buildRegistries(CoreDomains(), modules)
}

// buildRegistries executes registry bootstrap with explicit domain/module inputs.
// This seam keeps production wiring simple while allowing deterministic branch
// tests around startup validation failures.
func buildRegistries(domains []CoreDomain, modules []module.Module) (Registries, error) {
	bootstrap := newRegistryBootstrap(modules)

	if err := bootstrap.registerCoreDomains(domains); err != nil {
		return Registries{}, err
	}

	if err := bootstrap.validateCoreRegistrations(); err != nil {
		return Registries{}, err
	}

	if err := bootstrap.registerSystemModules(); err != nil {
		return Registries{}, err
	}

	if err := bootstrap.validateRegistryContracts(domains); err != nil {
		return Registries{}, err
	}

	if err := bootstrap.validateProjectionRegistries(domains); err != nil {
		return Registries{}, err
	}

	if err := bootstrap.validatePayloadValidators(); err != nil {
		return Registries{}, err
	}

	return Registries{
		Commands: bootstrap.commandRegistry,
		Events:   bootstrap.eventRegistry,
		Systems:  bootstrap.systemRegistry,
	}, nil
}

// registerCoreDomains keeps core registration isolated from system module
// wiring so command/event contract failures are reported with domain context.
func (b registryBootstrap) registerCoreDomains(domains []CoreDomain) error {
	for _, domain := range domains {
		if err := domain.RegisterCommands(b.commandRegistry); err != nil {
			return fmt.Errorf("register %s commands: %w", domain.Name(), err)
		}
		if err := domain.RegisterEvents(b.eventRegistry); err != nil {
			return fmt.Errorf("register %s events: %w", domain.Name(), err)
		}
	}
	return nil
}

// validateCoreRegistrations enforces core domain emission declarations against
// the shared event registry before system modules are loaded.
func (b registryBootstrap) validateCoreRegistrations() error {
	return validateCoreEmittableEventTypes(b.eventRegistry)
}

// registerSystemModules executes the module registration phase and validates
// new system-owned command/event types against namespace and emit declarations.
func (b registryBootstrap) registerSystemModules() error {
	for _, mod := range b.modules {
		if err := b.systemRegistry.Register(mod); err != nil {
			return err
		}
		beforeCommands := commandTypeSet(b.commandRegistry.ListDefinitions())
		if err := mod.RegisterCommands(b.commandRegistry); err != nil {
			return err
		}
		beforeEvents := eventTypeSet(b.eventRegistry.ListDefinitions())
		if err := mod.RegisterEvents(b.eventRegistry); err != nil {
			return err
		}
		if err := validateModuleSystemTypePrefixes(
			mod,
			beforeCommands,
			beforeEvents,
			b.commandRegistry.ListDefinitions(),
			b.eventRegistry.ListDefinitions(),
		); err != nil {
			return err
		}
		if err := validateEmittableEventTypes(mod, b.eventRegistry); err != nil {
			return err
		}
	}
	return nil
}

type registryValidationStep func(registryBootstrap) error

// runRegistryValidationPipeline runs registry validators in order and short
// circuits on first failure to preserve startup fail-fast behavior.
func runRegistryValidationPipeline(bootstrap registryBootstrap, steps ...registryValidationStep) error {
	for _, step := range steps {
		if err := step(bootstrap); err != nil {
			return err
		}
	}
	return nil
}

// validateRegistryContracts executes aggregate/runtime guardrails after all
// command/event definitions are loaded into registries.
func (b registryBootstrap) validateRegistryContracts(domains []CoreDomain) error {
	allFoldHandled := collectFoldHandledTypes(domains, b.modules)
	return runRegistryValidationPipeline(
		b,
		func(state registryBootstrap) error {
			return ValidateSystemReadinessCheckerCoverage(state.systemRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateSystemFoldCoverage(state.systemRegistry, state.eventRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateDeciderCommandCoverage(state.systemRegistry, state.commandRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateCoreDeciderCommandCoverage(state.commandRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateActiveSessionPolicyCoverage(state.commandRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateFoldCoverage(state.eventRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateAliasFoldCoverage(state.eventRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateAggregateFoldDispatch(state.eventRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateEntityKeyedAddressing(state.eventRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateNoFoldHandlersForAuditOnlyEvents(state.eventRegistry, allFoldHandled)
		},
		func(state registryBootstrap) error {
			return ValidateStateFactoryDeterminism(state.systemRegistry)
		},
		func(state registryBootstrap) error {
			return ValidateSystemMetadataConsistency(state.eventRegistry, state.systemRegistry)
		},
	)
}

// validateProjectionRegistries ensures projection declarations stay aligned with
// current registry definitions and module metadata.
func (b registryBootstrap) validateProjectionRegistries(domains []CoreDomain) error {
	projectionHandled := collectProjectionHandledTypes(domains)
	return ValidateProjectionRegistries(b.eventRegistry, b.systemRegistry, nil, projectionHandled)
}

// validatePayloadValidators rejects non-audit event types that skipped payload
// validation wiring so append-time contracts remain explicit.
func (b registryBootstrap) validatePayloadValidators() error {
	missing := b.eventRegistry.MissingPayloadValidators()
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, len(missing))
	for i, t := range missing {
		names[i] = string(t)
	}
	return fmt.Errorf("non-audit events without payload validators: %s", strings.Join(names, ", "))
}

// collectFoldHandledTypes gathers fold declarations across core domains and
// system modules for intent-guard validation.
func collectFoldHandledTypes(domains []CoreDomain, modules []module.Module) []event.Type {
	var allFoldHandled []event.Type
	for _, domain := range domains {
		allFoldHandled = append(allFoldHandled, domain.FoldHandledTypes()...)
	}
	for _, mod := range modules {
		if folder := mod.Folder(); folder != nil {
			allFoldHandled = append(allFoldHandled, folder.FoldHandledTypes()...)
		}
	}
	return allFoldHandled
}

// collectProjectionHandledTypes gathers core projection declarations used by
// projection registry consistency checks.
func collectProjectionHandledTypes(domains []CoreDomain) []event.Type {
	projectionHandled := make([]event.Type, 0)
	for _, domain := range domains {
		if domain.ProjectionHandledTypes != nil {
			projectionHandled = append(projectionHandled, domain.ProjectionHandledTypes()...)
		}
	}
	return projectionHandled
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
