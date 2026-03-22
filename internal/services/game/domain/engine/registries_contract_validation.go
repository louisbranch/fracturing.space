package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// registryContractValidator owns post-registration write-path and projection
// contract checks once core and system definitions have been loaded.
type registryContractValidator struct {
	domains []CoreDomain
	modules []module.Module
}

// newRegistryContractValidator captures the domain/module context needed for
// fold and projection coverage checks.
func newRegistryContractValidator(domains []CoreDomain, modules []module.Module) registryContractValidator {
	return registryContractValidator{
		domains: domains,
		modules: modules,
	}
}

// ValidateWritePath executes aggregate/runtime guardrails after all command and
// event definitions are loaded into registries.
func (v registryContractValidator) ValidateWritePath(bootstrap registryBootstrap) error {
	allFoldHandled := collectFoldHandledTypes(v.domains, v.modules)
	return runNamedRegistryValidationPipeline(
		bootstrap,
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
		namedRegistryValidationStep{name: "state factory fold compatibility", run: func(state registryBootstrap) error {
			return ValidateStateFactoryFoldCompatibility(state.systemRegistry)
		}},
		namedRegistryValidationStep{name: "system metadata consistency", run: func(state registryBootstrap) error {
			return ValidateSystemMetadataConsistency(state.eventRegistry, state.systemRegistry)
		}},
	)
}

// ValidateProjection ensures projection declarations stay aligned with current
// registry definitions and module metadata.
func (v registryContractValidator) ValidateProjection(bootstrap registryBootstrap) error {
	projectionHandled := collectProjectionHandledTypes(v.domains)
	return ValidateProjectionRegistries(bootstrap.eventRegistry, bootstrap.systemRegistry, nil, projectionHandled)
}

// registryPayloadValidator rejects non-audit event types that skipped payload
// validation wiring so append-time contracts remain explicit.
type registryPayloadValidator struct{}

// Validate checks that every non-audit event type has an append-time payload
// validator registered.
func (registryPayloadValidator) Validate(bootstrap registryBootstrap) error {
	missing := bootstrap.eventRegistry.MissingPayloadValidators()
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, len(missing))
	for i, t := range missing {
		names[i] = string(t)
	}
	return fmt.Errorf("non-audit events without payload validators: %s", strings.Join(names, ", "))
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

// validateRegistryContracts keeps the historical test seam while delegating the
// actual write-path validation responsibility to the contract validator.
func (b registryBootstrap) validateRegistryContracts(domains []CoreDomain) error {
	return newRegistryContractValidator(domains, b.modules).ValidateWritePath(b)
}

// validateProjectionRegistries keeps the historical test seam while delegating
// projection validation to the explicit contract validator collaborator.
func (b registryBootstrap) validateProjectionRegistries(domains []CoreDomain) error {
	return newRegistryContractValidator(domains, b.modules).ValidateProjection(b)
}

// validatePayloadValidators keeps the historical test seam while delegating the
// payload-validator contract check to a dedicated collaborator.
func (b registryBootstrap) validatePayloadValidators() error {
	return registryPayloadValidator{}.Validate(b)
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
