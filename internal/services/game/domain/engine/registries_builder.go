package engine

import (
	"fmt"
	"strings"

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

type registryValidationStep func(registryBootstrap) error

type namedRegistryValidationStep struct {
	name string
	run  registryValidationStep
}

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

// runNamedRegistryValidationPipeline behaves like runRegistryValidationPipeline
// but wraps failures with the explicit validation step name so startup errors
// are easier to triage.
func runNamedRegistryValidationPipeline(bootstrap registryBootstrap, steps ...namedRegistryValidationStep) error {
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		if err := step.run(bootstrap); err != nil {
			return fmt.Errorf("registry validation %s: %w", step.name, err)
		}
	}
	return nil
}

// validateRegistryContracts executes aggregate/runtime guardrails after all
// command/event definitions are loaded into registries.
func (b registryBootstrap) validateRegistryContracts(domains []CoreDomain) error {
	allFoldHandled := collectFoldHandledTypes(domains, b.modules)
	return runNamedRegistryValidationPipeline(
		b,
		namedRegistryValidationStep{name: "system readiness checker coverage", run: func(state registryBootstrap) error {
			return ValidateSystemReadinessCheckerCoverage(state.systemRegistry)
		}},
		namedRegistryValidationStep{name: "system fold coverage", run: func(state registryBootstrap) error {
			return ValidateSystemFoldCoverage(state.systemRegistry, state.eventRegistry)
		}},
		namedRegistryValidationStep{name: "system decider command coverage", run: func(state registryBootstrap) error {
			return ValidateDeciderCommandCoverage(state.systemRegistry, state.commandRegistry)
		}},
		namedRegistryValidationStep{name: "core decider command coverage", run: func(state registryBootstrap) error {
			return ValidateCoreDeciderCommandCoverage(state.commandRegistry)
		}},
		namedRegistryValidationStep{name: "active session policy coverage", run: func(state registryBootstrap) error {
			return ValidateActiveSessionPolicyCoverage(state.commandRegistry)
		}},
		namedRegistryValidationStep{name: "fold coverage", run: func(state registryBootstrap) error {
			return ValidateFoldCoverage(state.eventRegistry)
		}},
		namedRegistryValidationStep{name: "alias fold coverage", run: func(state registryBootstrap) error {
			return ValidateAliasFoldCoverage(state.eventRegistry)
		}},
		namedRegistryValidationStep{name: "aggregate fold dispatch", run: func(state registryBootstrap) error {
			return ValidateAggregateFoldDispatch(state.eventRegistry)
		}},
		namedRegistryValidationStep{name: "entity keyed addressing", run: func(state registryBootstrap) error {
			return ValidateEntityKeyedAddressing(state.eventRegistry)
		}},
		namedRegistryValidationStep{name: "audit-only fold handler exclusion", run: func(state registryBootstrap) error {
			return ValidateNoFoldHandlersForAuditOnlyEvents(state.eventRegistry, allFoldHandled)
		}},
		namedRegistryValidationStep{name: "state factory determinism", run: func(state registryBootstrap) error {
			return ValidateStateFactoryDeterminism(state.systemRegistry)
		}},
		namedRegistryValidationStep{name: "system metadata consistency", run: func(state registryBootstrap) error {
			return ValidateSystemMetadataConsistency(state.eventRegistry, state.systemRegistry)
		}},
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
