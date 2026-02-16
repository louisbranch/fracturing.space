package server

import (
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
	domainsystems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

var (
	errSystemMetadataRegistryRequired = errors.New("system metadata registry is required")
	errSystemAdapterRegistryRequired  = errors.New("system adapter registry is required")
	errSystemModuleRegistryMismatch   = errors.New("system module registry mismatch")
)

func registeredSystemModules() []domainsystem.Module {
	return systemmanifest.Modules()
}

func registeredMetadataSystems() []domainsystems.GameSystem {
	return systemmanifest.MetadataSystems()
}

func validateSystemRegistrationParity(modules []domainsystem.Module, metadata *domainsystems.Registry, adapters *domainsystems.AdapterRegistry) error {
	if metadata == nil {
		return errSystemMetadataRegistryRequired
	}
	if adapters == nil {
		return errSystemAdapterRegistryRequired
	}

	moduleKeys := make(map[string]struct{}, len(modules))
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
		moduleKeys[systemParityKey(gameSystem, moduleVersion)] = struct{}{}
		if metadata.GetVersion(gameSystem, moduleVersion) == nil {
			return fmt.Errorf("%w: metadata missing for module %s@%s", errSystemModuleRegistryMismatch, moduleID, moduleVersion)
		}
		if adapters.Get(gameSystem, moduleVersion) == nil {
			return fmt.Errorf("%w: adapter missing for module %s@%s", errSystemModuleRegistryMismatch, moduleID, moduleVersion)
		}
	}

	for _, gameSystem := range metadata.List() {
		if gameSystem == nil {
			continue
		}
		version := strings.TrimSpace(gameSystem.Version())
		key := systemParityKey(gameSystem.ID(), version)
		if _, ok := moduleKeys[key]; !ok {
			return fmt.Errorf("%w: metadata registered without module %s@%s", errSystemModuleRegistryMismatch, gameSystem.ID(), version)
		}
		if adapters.Get(gameSystem.ID(), version) == nil {
			return fmt.Errorf("%w: adapter missing for metadata %s@%s", errSystemModuleRegistryMismatch, gameSystem.ID(), version)
		}
	}
	return nil
}

func parseGameSystemID(raw string) (commonv1.GameSystem, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("system id is required")
	}
	if value, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(value), nil
	}
	upper := strings.ToUpper(trimmed)
	if value, ok := commonv1.GameSystem_value["GAME_SYSTEM_"+upper]; ok {
		return commonv1.GameSystem(value), nil
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unknown system id: %s", trimmed)
}

func systemParityKey(id commonv1.GameSystem, version string) string {
	return fmt.Sprintf("%d@%s", id, strings.TrimSpace(version))
}
