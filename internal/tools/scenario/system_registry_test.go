package scenario

import (
	"context"
	"strings"
	"testing"

	"github.com/Shopify/go-lua"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestValidateScenarioSystemRegistry(t *testing.T) {
	t.Run("valid registry", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART),
		}
		if err := validateScenarioSystemRegistry(registry); err != nil {
			t.Fatalf("validateScenarioSystemRegistry: %v", err)
		}
	})

	t.Run("requires explicit id", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": func() scenarioSystemRegistration {
				registration := validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
				registration.id = ""
				return registration
			}(),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "empty id") {
			t.Fatalf("expected empty id validation error, got %v", err)
		}
	})

	t.Run("rejects unspecified game system", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "GAME_SYSTEM_UNSPECIFIED") {
			t.Fatalf("expected unspecified game system error, got %v", err)
		}
	})

	t.Run("requires step runner", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": func() scenarioSystemRegistration {
				registration := validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
				registration.runStep = nil
				return registration
			}(),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "step runner") {
			t.Fatalf("expected step runner error, got %v", err)
		}
	})

	t.Run("requires non-empty step kinds", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": func() scenarioSystemRegistration {
				registration := validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
				registration.stepKinds = map[string]struct{}{}
				return registration
			}(),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "step kind") {
			t.Fatalf("expected step kind validation error, got %v", err)
		}
	})

	t.Run("rejects duplicate method names in one system", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": func() scenarioSystemRegistration {
				registration := validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
				registration.dslMethods = append(registration.dslMethods, lua.RegistryFunction{
					Name:     registration.dslMethods[0].Name,
					Function: stubLuaMethod,
				})
				return registration
			}(),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "duplicated in system") {
			t.Fatalf("expected duplicate method validation error, got %v", err)
		}
	})

	t.Run("rejects unknown game system value", func(t *testing.T) {
		registry := map[string]scenarioSystemRegistration{
			"DAGGERHEART": validScenarioSystemRegistration("DAGGERHEART", commonv1.GameSystem(42)),
		}
		err := validateScenarioSystemRegistry(registry)
		if err == nil || !strings.Contains(err.Error(), "unknown game system value") {
			t.Fatalf("expected unknown game system error, got %v", err)
		}
	})
}

func validScenarioSystemRegistration(id string, gameSystem commonv1.GameSystem) scenarioSystemRegistration {
	methodName := strings.ToLower(id) + "_method"
	stepKind := strings.ToLower(id) + "_step"
	return scenarioSystemRegistration{
		id:         id,
		gameSystem: gameSystem,
		dslMethods: []lua.RegistryFunction{
			{Name: methodName, Function: stubLuaMethod},
		},
		stepKinds: map[string]struct{}{
			stepKind: {},
		},
		runStep: stubSystemStepRunner,
	}
}

func stubSystemStepRunner(_ *Runner, _ context.Context, _ *scenarioState, _ Step) error {
	return nil
}

func stubLuaMethod(_ *lua.State) int {
	return 0
}
