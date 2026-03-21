package engine

import (
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

// registries returns the assembled public registry bundle once bootstrap
// registration and validation have succeeded.
func (b registryBootstrap) registries() Registries {
	return Registries{
		Commands: b.commandRegistry,
		Events:   b.eventRegistry,
		Systems:  b.systemRegistry,
	}
}

type registryBuildWorkflow struct {
	bootstrap         registryBootstrap
	coreDomains       registryCoreDomainRegistrar
	systemModules     registrySystemModuleRegistrar
	contractValidator registryContractValidator
	payloadValidator  registryPayloadValidator
}

// newRegistryBuildWorkflow assembles the explicit collaborators that own the
// registry bootstrap phases used during startup.
func newRegistryBuildWorkflow(domains []CoreDomain, modules []module.Module) registryBuildWorkflow {
	return registryBuildWorkflow{
		bootstrap:         newRegistryBootstrap(modules),
		coreDomains:       newRegistryCoreDomainRegistrar(domains),
		systemModules:     registrySystemModuleRegistrar{},
		contractValidator: newRegistryContractValidator(domains, modules),
		payloadValidator:  registryPayloadValidator{},
	}
}

// Build executes core registration, system registration, and contract
// validation in the canonical startup order before returning the assembled
// registries.
func (w registryBuildWorkflow) Build() (Registries, error) {
	if err := w.coreDomains.Register(w.bootstrap); err != nil {
		return Registries{}, err
	}
	if err := w.coreDomains.Validate(w.bootstrap); err != nil {
		return Registries{}, err
	}
	if err := w.systemModules.Register(w.bootstrap); err != nil {
		return Registries{}, err
	}
	if err := w.contractValidator.ValidateWritePath(w.bootstrap); err != nil {
		return Registries{}, err
	}
	if err := w.contractValidator.ValidateProjection(w.bootstrap); err != nil {
		return Registries{}, err
	}
	if err := w.payloadValidator.Validate(w.bootstrap); err != nil {
		return Registries{}, err
	}
	return w.bootstrap.registries(), nil
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
	return newRegistryBuildWorkflow(domains, modules).Build()
}
