//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestScenarioMissingMechanicTimelineCoverage(t *testing.T) {
	repoRoot := integrationRepoRoot(t)
	scenarioDir := filepath.Join(repoRoot, "internal", "test", "game", "scenarios")
	missingMechanicsDoc := filepath.Join(repoRoot, "docs", "project", "scenario-missing-mechanics.md")
	timelineDoc := filepath.Join(repoRoot, "docs", "project", "daggerheart-event-timeline-contract.md")

	markerScenarios, err := loadMarkedScenarioFiles(scenarioDir, "-- Missing Mechanic:")
	if err != nil {
		t.Fatalf("load marker scenarios: %v", err)
	}

	indexRows, err := loadScenarioTimelineIndex(missingMechanicsDoc)
	if err != nil {
		t.Fatalf("load scenario timeline index: %v", err)
	}
	if len(indexRows) == 0 {
		t.Fatal("expected at least one scenario timeline mapping")
	}

	timelineRowIDs, err := loadTimelineRowIDs(timelineDoc)
	if err != nil {
		t.Fatalf("load timeline row ids: %v", err)
	}
	if len(timelineRowIDs) == 0 {
		t.Fatal("expected at least one timeline row id")
	}

	if err := validateTimelineCoverageForMarkers(markerScenarios, indexRows, timelineRowIDs); err != nil {
		t.Fatal(err)
	}
}

func TestDaggerheartTimelineTypesAreRegistered(t *testing.T) {
	repoRoot := integrationRepoRoot(t)
	timelineDoc := filepath.Join(repoRoot, "docs", "project", "daggerheart-event-timeline-contract.md")

	commandTypes, eventTypes, err := loadTimelineCommandAndEventTypes(timelineDoc)
	if err != nil {
		t.Fatalf("load timeline command/event types: %v", err)
	}
	if len(commandTypes) == 0 {
		t.Fatal("expected at least one command type in timeline contract")
	}
	if len(eventTypes) == 0 {
		t.Fatal("expected at least one event type in timeline contract")
	}

	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	commandDefs := registries.Commands.ListDefinitions()
	knownCommands := make(map[string]struct{}, len(commandDefs))
	for _, definition := range commandDefs {
		knownCommands[string(definition.Type)] = struct{}{}
	}
	for commandType := range commandTypes {
		if _, ok := knownCommands[commandType]; !ok {
			t.Fatalf("timeline command type %s is not registered", commandType)
		}
	}

	eventDefs := registries.Events.ListDefinitions()
	knownEvents := make(map[string]struct{}, len(eventDefs))
	for _, definition := range eventDefs {
		knownEvents[string(definition.Type)] = struct{}{}
	}
	for eventType := range eventTypes {
		if _, ok := knownEvents[eventType]; !ok {
			t.Fatalf("timeline event type %s is not registered", eventType)
		}
	}
}
