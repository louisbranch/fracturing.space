package scenario

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Shopify/go-lua"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

type systemStepRunner func(r *Runner, ctx context.Context, state *scenarioState, step Step) error

type systemCharacterReadinessRunner func(r *Runner, ctx context.Context, state *scenarioState, characterID string) error

type systemCharacterNeedsReadinessRunner func(r *Runner, ctx context.Context, state *scenarioState, characterID string) (bool, error)

type scenarioSystemRegistration struct {
	id                       string
	gameSystem               commonv1.GameSystem
	dslMethods               []lua.RegistryFunction
	stepKinds                map[string]struct{}
	runStep                  systemStepRunner
	ensureCharacterReadiness systemCharacterReadinessRunner
	characterNeedsReadiness  systemCharacterNeedsReadinessRunner
}

var (
	scenarioSystemRegistryOnce       sync.Once
	scenarioSystemRegistryInitErr    error
	scenarioSystemIDsSorted          []string
	scenarioSystemByID               map[string]scenarioSystemRegistration
	scenarioSystemByGameSystem       map[commonv1.GameSystem]scenarioSystemRegistration
	scenarioSystemHandleMethods      []lua.RegistryFunction
	scenarioSystemStepKindsToSystems map[string][]string
	scenarioSystemKnownStepKinds     map[string]struct{}
)

var sharedScenarioStepKinds = map[string]struct{}{}

func indexScenarioSystemRegistry() {
	scenarioSystemRegistryOnce.Do(func() {
		normalizedRegistry := make(map[string]scenarioSystemRegistration, len(scenarioSystemRegistry))
		ids := make([]string, 0, len(scenarioSystemRegistry))
		for id, registration := range scenarioSystemRegistry {
			normalizedID := strings.ToUpper(strings.TrimSpace(id))
			if normalizedID == "" {
				continue
			}
			if _, exists := normalizedRegistry[normalizedID]; exists {
				scenarioSystemRegistryInitErr = fmt.Errorf("duplicate key %q after normalization", normalizedID)
				return
			}
			registration = bindScenarioSystemRuntimeHooks(registration)
			normalizedRegistry[normalizedID] = registration
			ids = append(ids, normalizedID)
		}
		if err := validateScenarioSystemRegistry(normalizedRegistry); err != nil {
			scenarioSystemRegistryInitErr = err
			return
		}
		sort.Strings(ids)
		scenarioSystemIDsSorted = ids

		scenarioSystemByID = make(map[string]scenarioSystemRegistration, len(ids))
		scenarioSystemByGameSystem = make(map[commonv1.GameSystem]scenarioSystemRegistration, len(ids))
		scenarioSystemStepKindsToSystems = map[string][]string{}
		scenarioSystemKnownStepKinds = map[string]struct{}{}
		methodNames := map[string]struct{}{}

		for _, id := range ids {
			registration := normalizedRegistry[id]
			scenarioSystemByID[registration.id] = registration
			scenarioSystemByGameSystem[registration.gameSystem] = registration

			for _, method := range registration.dslMethods {
				if strings.TrimSpace(method.Name) == "" {
					continue
				}
				// System handles share a metatable, so keep one method entry per name.
				if _, exists := methodNames[method.Name]; exists {
					continue
				}
				methodNames[method.Name] = struct{}{}
				scenarioSystemHandleMethods = append(scenarioSystemHandleMethods, method)
			}
			for stepKind := range registration.stepKinds {
				if strings.TrimSpace(stepKind) == "" {
					continue
				}
				scenarioSystemKnownStepKinds[stepKind] = struct{}{}
				scenarioSystemStepKindsToSystems[stepKind] = append(scenarioSystemStepKindsToSystems[stepKind], registration.id)
			}
		}
		for stepKind := range scenarioSystemStepKindsToSystems {
			sort.Strings(scenarioSystemStepKindsToSystems[stepKind])
		}
	})
}

func ensureScenarioSystemRegistry() error {
	indexScenarioSystemRegistry()
	if scenarioSystemRegistryInitErr != nil {
		return fmt.Errorf("invalid scenario system registry: %w", scenarioSystemRegistryInitErr)
	}
	return nil
}

func registeredScenarioSystemIDs() []string {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return nil
	}
	return append([]string(nil), scenarioSystemIDsSorted...)
}

func registeredScenarioSystemMethods() []lua.RegistryFunction {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return nil
	}
	return append([]lua.RegistryFunction(nil), scenarioSystemHandleMethods...)
}

func scenarioSystemForID(systemID string) (scenarioSystemRegistration, bool, error) {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return scenarioSystemRegistration{}, false, err
	}
	normalized := strings.ToUpper(strings.TrimSpace(systemID))
	registration, ok := scenarioSystemByID[normalized]
	return registration, ok, nil
}

func scenarioSystemForGameSystem(system commonv1.GameSystem) (scenarioSystemRegistration, bool, error) {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return scenarioSystemRegistration{}, false, err
	}
	registration, ok := scenarioSystemByGameSystem[system]
	return registration, ok, nil
}

func isKnownScenarioSystemStepKind(stepKind string) (bool, error) {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return false, err
	}
	_, ok := scenarioSystemKnownStepKinds[stepKind]
	return ok, nil
}

func registeredSystemsForStepKind(stepKind string) ([]string, error) {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return nil, err
	}
	systems := scenarioSystemStepKindsToSystems[stepKind]
	return append([]string(nil), systems...), nil
}

func registeredScenarioSystemIDForGameSystem(system commonv1.GameSystem) (string, error) {
	registration, ok, err := scenarioSystemForGameSystem(system)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", unsupportedScenarioSystemError(strings.TrimPrefix(system.String(), "GAME_SYSTEM_"))
	}
	return registration.id, nil
}

func registeredScenarioSystemIDForValue(value string) (string, commonv1.GameSystem, error) {
	gameSystem, err := parseGameSystem(value)
	if err != nil {
		return "", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, err
	}
	systemID, err := registeredScenarioSystemIDForGameSystem(gameSystem)
	if err != nil {
		return "", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, err
	}
	return systemID, gameSystem, nil
}

func scenarioSystemForState(state *scenarioState) (scenarioSystemRegistration, bool, error) {
	if state == nil || state.campaignSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return scenarioSystemRegistration{}, false, nil
	}
	registration, ok, err := scenarioSystemForGameSystem(state.campaignSystem)
	if err != nil {
		return scenarioSystemRegistration{}, false, err
	}
	if !ok {
		return scenarioSystemRegistration{}, false, unsupportedScenarioSystemError(strings.TrimPrefix(state.campaignSystem.String(), "GAME_SYSTEM_"))
	}
	return registration, true, nil
}

func unsupportedScenarioSystemError(value string) error {
	if err := ensureScenarioSystemRegistry(); err != nil {
		return err
	}
	normalized := strings.ToUpper(strings.TrimSpace(value))
	registered := registeredScenarioSystemIDs()
	if len(registered) == 0 {
		return fmt.Errorf("unsupported scenario system %q", normalized)
	}
	return fmt.Errorf("unsupported scenario system %q (registered systems: %s)", normalized, strings.Join(registered, ", "))
}

func bindScenarioSystemRuntimeHooks(registration scenarioSystemRegistration) scenarioSystemRegistration {
	switch registration.id {
	case "DAGGERHEART":
		registration.runStep = (*Runner).runDaggerheartStep
		registration.ensureCharacterReadiness = (*Runner).ensureDaggerheartCharacterReadiness
		registration.characterNeedsReadiness = (*Runner).daggerheartCharacterNeedsReadiness
	}
	return registration
}

func validateScenarioSystemRegistry(registry map[string]scenarioSystemRegistration) error {
	if len(registry) == 0 {
		return fmt.Errorf("scenario system registry is empty")
	}

	seenIDs := make(map[string]struct{}, len(registry))
	seenGameSystems := make(map[commonv1.GameSystem]string, len(registry))
	seenMethodOwners := map[string]string{}
	seenStepKindOwners := map[string]string{}

	for key, registration := range registry {
		normalizedKey := strings.ToUpper(strings.TrimSpace(key))
		if normalizedKey == "" {
			return fmt.Errorf("scenario system registry contains empty key")
		}

		normalizedID := strings.ToUpper(strings.TrimSpace(registration.id))
		if normalizedID == "" {
			return fmt.Errorf("scenario system %q has empty id", normalizedKey)
		}
		if normalizedID != normalizedKey {
			return fmt.Errorf("scenario system key %q must match id %q", normalizedKey, normalizedID)
		}
		if _, exists := seenIDs[normalizedID]; exists {
			return fmt.Errorf("scenario system id %q is duplicated", normalizedID)
		}
		seenIDs[normalizedID] = struct{}{}

		if registration.gameSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
			return fmt.Errorf("scenario system %q must not use GAME_SYSTEM_UNSPECIFIED", normalizedID)
		}
		if _, exists := commonv1.GameSystem_name[int32(registration.gameSystem)]; !exists {
			return fmt.Errorf("scenario system %q uses unknown game system value %d", normalizedID, int32(registration.gameSystem))
		}
		if owner, exists := seenGameSystems[registration.gameSystem]; exists && owner != normalizedID {
			return fmt.Errorf("game system %q is registered by multiple scenario systems (%s, %s)", registration.gameSystem.String(), owner, normalizedID)
		}
		seenGameSystems[registration.gameSystem] = normalizedID

		if len(registration.dslMethods) == 0 {
			return fmt.Errorf("scenario system %q must define at least one DSL method", normalizedID)
		}
		for _, method := range registration.dslMethods {
			name := strings.TrimSpace(method.Name)
			if name == "" {
				return fmt.Errorf("scenario system %q has an empty DSL method name", normalizedID)
			}
			if method.Function == nil {
				return fmt.Errorf("scenario system %q has nil DSL method function for %q", normalizedID, name)
			}
			if owner, exists := seenMethodOwners[name]; exists {
				if owner == normalizedID {
					return fmt.Errorf("scenario DSL method %q is duplicated in system %s", name, normalizedID)
				}
				return fmt.Errorf("scenario DSL method %q is registered by multiple systems (%s, %s)", name, owner, normalizedID)
			}
			seenMethodOwners[name] = normalizedID
		}

		if registration.runStep == nil {
			return fmt.Errorf("scenario system %q must define a step runner", normalizedID)
		}
		if len(registration.stepKinds) == 0 {
			return fmt.Errorf("scenario system %q must define at least one step kind", normalizedID)
		}
		for stepKind := range registration.stepKinds {
			normalizedStepKind := strings.TrimSpace(stepKind)
			if normalizedStepKind == "" {
				return fmt.Errorf("scenario system %q has empty step kind", normalizedID)
			}
			if owner, exists := seenStepKindOwners[normalizedStepKind]; exists && owner != normalizedID {
				if _, allowed := sharedScenarioStepKinds[normalizedStepKind]; !allowed {
					return fmt.Errorf("scenario step kind %q is registered by multiple systems (%s, %s)", normalizedStepKind, owner, normalizedID)
				}
			}
			seenStepKindOwners[normalizedStepKind] = normalizedID
		}
	}

	return nil
}
