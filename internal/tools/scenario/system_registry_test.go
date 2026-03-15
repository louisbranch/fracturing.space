package scenario

import (
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestRegisteredScenarioSystemIDsReturnsNormalizedList(t *testing.T) {
	t.Parallel()

	ids := registeredScenarioSystemIDs()
	if len(ids) == 0 {
		t.Fatalf("registeredScenarioSystemIDs() returned no ids")
	}
	if ids[0] != "DAGGERHEART" {
		t.Fatalf("registeredScenarioSystemIDs()[0] = %q, want DAGGERHEART", ids[0])
	}
}

func TestRegisteredScenarioSystemMethodsReturnsKnownDaggerheartHandle(t *testing.T) {
	t.Parallel()

	methods := registeredScenarioSystemMethods()
	if len(methods) == 0 {
		t.Fatalf("registeredScenarioSystemMethods() returned no methods")
	}
	found := false
	for _, method := range methods {
		if method.Name == "action_roll" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("registeredScenarioSystemMethods() did not include action_roll")
	}
}

func TestRegisteredSystemsForStepKindAndKnownStepKinds(t *testing.T) {
	t.Parallel()

	ok, err := isKnownScenarioSystemStepKind("action_roll")
	if err != nil {
		t.Fatalf("isKnownScenarioSystemStepKind() error = %v", err)
	}
	if !ok {
		t.Fatalf("isKnownScenarioSystemStepKind(action_roll) = false, want true")
	}

	systems, err := registeredSystemsForStepKind("action_roll")
	if err != nil {
		t.Fatalf("registeredSystemsForStepKind() error = %v", err)
	}
	if len(systems) != 1 || systems[0] != "DAGGERHEART" {
		t.Fatalf("registeredSystemsForStepKind(action_roll) = %v, want [DAGGERHEART]", systems)
	}
}

func TestRegisteredScenarioSystemIDHelpers(t *testing.T) {
	t.Parallel()

	got, err := registeredScenarioSystemIDForGameSystem(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	if err != nil {
		t.Fatalf("registeredScenarioSystemIDForGameSystem() error = %v", err)
	}
	if got != "DAGGERHEART" {
		t.Fatalf("registeredScenarioSystemIDForGameSystem() = %q, want DAGGERHEART", got)
	}

	systemID, gameSystem, err := registeredScenarioSystemIDForValue("daggerheart")
	if err != nil {
		t.Fatalf("registeredScenarioSystemIDForValue() error = %v", err)
	}
	if systemID != "DAGGERHEART" || gameSystem != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("registeredScenarioSystemIDForValue() = (%q, %v), want (DAGGERHEART, DAGGERHEART)", systemID, gameSystem)
	}
}

func TestScenarioSystemForStateAndUnsupportedError(t *testing.T) {
	t.Parallel()

	registration, ok, err := scenarioSystemForState(&scenarioState{campaignSystem: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART})
	if err != nil {
		t.Fatalf("scenarioSystemForState() error = %v", err)
	}
	if !ok || registration.id != "DAGGERHEART" {
		t.Fatalf("scenarioSystemForState() = (%+v, %v), want DAGGERHEART/true", registration, ok)
	}

	err = unsupportedScenarioSystemError("missing")
	if err == nil || !strings.Contains(err.Error(), "registered systems: DAGGERHEART") {
		t.Fatalf("unsupportedScenarioSystemError() = %v, want registered systems hint", err)
	}
}
