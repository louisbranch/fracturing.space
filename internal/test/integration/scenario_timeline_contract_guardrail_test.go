//go:build integration

package integration

import (
	"path/filepath"
	"testing"
)

func TestScenarioMissingMechanicTimelineCoverage(t *testing.T) {
	repoRoot := integrationRepoRoot(t)
	scenarioDir := filepath.Join(repoRoot, "internal", "test", "game", "scenarios")
	missingMechanicsDoc := filepath.Join(repoRoot, "docs", "project", "scenario-missing-mechanics.md")
	timelineDoc := filepath.Join(repoRoot, "docs", "project", "daggerheart-event-timeline-contract.md")

	markerScenarios, err := loadMarkedScenarioFiles(scenarioDir, "-- Missing DSL:")
	if err != nil {
		t.Fatalf("load marker scenarios: %v", err)
	}
	if len(markerScenarios) == 0 {
		t.Fatal("expected at least one scenario with -- Missing DSL marker")
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

	for scenario := range markerScenarios {
		rowID, ok := indexRows[scenario]
		if !ok {
			t.Fatalf("missing scenario timeline index row for %s", scenario)
		}
		if _, ok := timelineRowIDs[rowID]; !ok {
			t.Fatalf("scenario %s maps to unknown timeline row id %s", scenario, rowID)
		}
	}
}
