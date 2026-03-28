package app

import (
	"errors"
	"fmt"
	"strings"

	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

var (
	errSystemMetadataRegistryRequired = errors.New("system metadata registry is required")
	errSystemAdapterRegistryRequired  = errors.New("system adapter registry is required")
	errSystemModuleRegistryMismatch   = errors.New("system module registry mismatch")
)

// systemRegistrationSnapshot freezes the manifest-derived system inventory used
// by one server bootstrap run.
//
// Startup should resolve the built-in module and metadata sets once, then pass
// that explicit snapshot through the registration and parity-validation phases
// instead of reaching back into manifest helpers from scattered call sites.
type systemRegistrationSnapshot struct {
	modules  []domainsystem.Module
	metadata []domainbridge.GameSystem
}

// loadSystemRegistrationSnapshot materializes the manifest-derived startup
// inventory once so later phases all validate against the same slice contents.
func loadSystemRegistrationSnapshot() systemRegistrationSnapshot {
	return systemRegistrationSnapshot{
		modules:  append([]domainsystem.Module(nil), systemmanifest.Modules()...),
		metadata: append([]domainbridge.GameSystem(nil), systemmanifest.MetadataSystems()...),
	}
}

// modulesCopy returns the frozen built-in system modules for this bootstrap run.
func (s systemRegistrationSnapshot) modulesCopy() []domainsystem.Module {
	return append([]domainsystem.Module(nil), s.modules...)
}

// metadataSystemsCopy returns the frozen built-in metadata registrations for this
// bootstrap run.
func (s systemRegistrationSnapshot) metadataSystemsCopy() []domainbridge.GameSystem {
	return append([]domainbridge.GameSystem(nil), s.metadata...)
}

// buildMetadataRegistry builds a registry from the frozen manifest metadata so
// startup parity checks and API registration observe one explicit inventory.
func (s systemRegistrationSnapshot) buildMetadataRegistry() (*domainbridge.MetadataRegistry, error) {
	registry := domainbridge.NewMetadataRegistry()
	for _, gameSystem := range s.metadata {
		if err := registry.Register(gameSystem); err != nil {
			return nil, fmt.Errorf("register system %s@%s: %w", gameSystem.ID(), gameSystem.Version(), err)
		}
	}
	return registry, nil
}

// validateSystemRegistrationParity ensures module, metadata, and adapter registries match.
//
// If a module is missing from either metadata or adapters (or vice versa), the
// server refuses startup because command execution and read-model projection would
// diverge by system.
func validateSystemRegistrationParity(modules []domainsystem.Module, metadata *domainbridge.MetadataRegistry, adapters *domainbridge.AdapterRegistry) error {
	if metadata == nil {
		return errSystemMetadataRegistryRequired
	}
	if adapters == nil {
		return errSystemAdapterRegistryRequired
	}

	moduleKeys := make(map[string]struct{}, len(modules))
	// systemIDToModuleID maps domain system IDs to module IDs so the metadata
	// loop can look up string-keyed adapters.
	systemIDToModuleID := make(map[domainbridge.SystemID]string, len(modules))
	for _, module := range modules {
		if module == nil {
			return fmt.Errorf("%w: module is nil", errSystemModuleRegistryMismatch)
		}
		moduleID := strings.TrimSpace(module.ID())
		moduleVersion := strings.TrimSpace(module.Version())
		if moduleID == "" || moduleVersion == "" {
			return fmt.Errorf("%w: module id/version is required", errSystemModuleRegistryMismatch)
		}
		gameSystem, err := parseGameSystemID(moduleID)
		if err != nil {
			return fmt.Errorf("%w: %w", errSystemModuleRegistryMismatch, err)
		}
		moduleKeys[systemParityKey(moduleID, moduleVersion)] = struct{}{}
		systemIDToModuleID[gameSystem] = moduleID
		if metadata.GetVersion(gameSystem, moduleVersion) == nil {
			return fmt.Errorf("%w: metadata missing for module %s@%s", errSystemModuleRegistryMismatch, moduleID, moduleVersion)
		}
		if !adapters.Has(moduleID, moduleVersion) {
			return fmt.Errorf("%w: adapter missing for module %s@%s", errSystemModuleRegistryMismatch, moduleID, moduleVersion)
		}
	}

	for _, gameSystem := range metadata.List() {
		if gameSystem == nil {
			continue
		}
		version := strings.TrimSpace(gameSystem.Version())
		moduleID, ok := systemIDToModuleID[gameSystem.ID()]
		if !ok {
			return fmt.Errorf("%w: metadata registered without module %s@%s", errSystemModuleRegistryMismatch, gameSystem.ID(), version)
		}
		key := systemParityKey(moduleID, version)
		if _, ok := moduleKeys[key]; !ok {
			return fmt.Errorf("%w: metadata registered without module %s@%s", errSystemModuleRegistryMismatch, gameSystem.ID(), version)
		}
		if !adapters.Has(moduleID, version) {
			return fmt.Errorf("%w: adapter missing for metadata %s@%s", errSystemModuleRegistryMismatch, gameSystem.ID(), version)
		}
	}
	return nil
}

// parseGameSystemID canonicalizes environment-facing system labels.
func parseGameSystemID(raw string) (domainbridge.SystemID, error) {
	systemID, ok := domainbridge.NormalizeSystemID(raw)
	if !ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return domainbridge.SystemIDUnspecified, fmt.Errorf("system id is required")
		}
		return domainbridge.SystemIDUnspecified, fmt.Errorf("unknown system id: %s", trimmed)
	}
	return systemID, nil
}

// systemParityKey normalizes system+version into a single key for cross-registry comparison.
func systemParityKey(id string, version string) string {
	return id + "@" + strings.TrimSpace(version)
}
