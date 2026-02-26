//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

func TestDaggerheartTimelineTypesAreRegistered(t *testing.T) {
	repoRoot := integrationRepoRoot(t)
	timelineDoc := filepath.Join(repoRoot, "docs", "architecture", "daggerheart-event-timeline-contract.md")

	commandTypes, eventTypes, err := loadTimelineCommandAndEventTypesFromDocs(timelineDoc)
	if err != nil {
		t.Fatalf("load timeline command/event types: %v", err)
	}
	if len(commandTypes) == 0 {
		t.Fatal("expected at least one command type in timeline docs")
	}
	if len(eventTypes) == 0 {
		t.Fatal("expected at least one event type in timeline docs")
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

func TestRegisteredDaggerheartTimelineTypesAreDocumented(t *testing.T) {
	repoRoot := integrationRepoRoot(t)
	timelineDoc := filepath.Join(repoRoot, "docs", "architecture", "daggerheart-event-timeline-contract.md")

	commandTypes, eventTypes, err := loadTimelineCommandAndEventTypesFromDocs(timelineDoc)
	if err != nil {
		t.Fatalf("load timeline command/event types: %v", err)
	}

	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	missingCommands := missingDaggerheartTimelineCommandTypes(commandTypes, registries.Commands.ListDefinitions())
	if len(missingCommands) > 0 {
		t.Fatalf("registered timeline-tracked command types missing from timeline docs: %v", missingCommands)
	}

	missingEvents := missingDaggerheartTimelineEventTypes(eventTypes, registries.Events.ListDefinitions())
	if len(missingEvents) > 0 {
		t.Fatalf("registered timeline-tracked event types missing from timeline docs: %v", missingEvents)
	}
}
