package engine

import (
	"fmt"
)

// registryCoreDomainRegistrar owns registration and baseline validation for
// core command/event contracts before system modules are loaded.
type registryCoreDomainRegistrar struct {
	domains []CoreDomain
}

// newRegistryCoreDomainRegistrar captures the core domains once so startup
// orchestration stays focused on phase order rather than domain iteration.
func newRegistryCoreDomainRegistrar(domains []CoreDomain) registryCoreDomainRegistrar {
	return registryCoreDomainRegistrar{domains: domains}
}

// Register loads all core command and event definitions into the shared
// registries before any system modules mutate them.
func (r registryCoreDomainRegistrar) Register(bootstrap registryBootstrap) error {
	for _, domain := range r.domains {
		if err := domain.RegisterCommands(bootstrap.commandRegistry); err != nil {
			return fmt.Errorf("register %s commands: %w", domain.Name(), err)
		}
		if err := domain.RegisterEvents(bootstrap.eventRegistry); err != nil {
			return fmt.Errorf("register %s events: %w", domain.Name(), err)
		}
	}
	return nil
}

// Validate enforces core domain emission declarations before system modules are
// added to the write-path registries.
func (r registryCoreDomainRegistrar) Validate(bootstrap registryBootstrap) error {
	return validateCoreEmittableEventTypes(bootstrap.eventRegistry, r.domains)
}

// registerCoreDomains keeps the historical test seam while delegating ownership
// to the dedicated core-domain registrar.
func (b registryBootstrap) registerCoreDomains(domains []CoreDomain) error {
	return newRegistryCoreDomainRegistrar(domains).Register(b)
}

// validateCoreRegistrations keeps the historical test seam while delegating the
// actual validation responsibility to the core-domain registrar.
func (b registryBootstrap) validateCoreRegistrations(domains []CoreDomain) error {
	return newRegistryCoreDomainRegistrar(domains).Validate(b)
}
