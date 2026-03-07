package server

import (
	"errors"
	"fmt"
	"strings"

	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

var (
	errSystemMetadataRegistryRequired = errors.New("system metadata registry is required")
	errSystemAdapterRegistryRequired  = errors.New("system adapter registry is required")
	errSystemModuleRegistryMismatch   = errors.New("system module registry mismatch")
)

// registeredSystemModules returns the concrete system implementations wired into runtime.
//
// These modules provide command/event registration plus adapters used by domain and
// projection code paths; keeping this in one place ensures startup can validate
// consistency before accepting traffic.
func registeredSystemModules() []domainsystem.Module {
	return systemmanifest.Modules()
}

// registeredMetadataSystems returns system metadata surfaced in API contracts and registry.
//
// The metadata side is the contract-level source of truth for system names and
// versions before runtime adapters are loaded.
func registeredMetadataSystems() []domainbridge.GameSystem {
	return systemmanifest.MetadataSystems()
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
			return fmt.Errorf("%w: %v", errSystemModuleRegistryMismatch, err)
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

	if err := systemmanifest.ValidateProfileAdapterCoverage(adapters); err != nil {
		return fmt.Errorf("%w: %v", errSystemModuleRegistryMismatch, err)
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
