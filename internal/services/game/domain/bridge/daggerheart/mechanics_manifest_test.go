package daggerheart

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestManifest_MechanicIDsAreUnique(t *testing.T) {
	manifest := MechanicsManifest()
	seen := make(map[string]struct{}, len(manifest))
	for _, m := range manifest {
		if _, dup := seen[m.ID]; dup {
			t.Fatalf("duplicate mechanic ID: %s", m.ID)
		}
		seen[m.ID] = struct{}{}
	}
}

func TestManifest_AllMechanicsHaveRequiredFields(t *testing.T) {
	for _, m := range MechanicsManifest() {
		if m.ID == "" {
			t.Fatal("mechanic with empty ID")
		}
		if m.Name == "" {
			t.Fatalf("mechanic %s has empty Name", m.ID)
		}
	}
}

func TestManifest_ImplementedCommandsAreRegistered(t *testing.T) {
	registered := make(map[command.Type]struct{}, len(daggerheartCommandDefinitions))
	for _, def := range daggerheartCommandDefinitions {
		registered[def.Type] = struct{}{}
	}

	for _, m := range MechanicsManifest() {
		if m.Status != MechanicImplemented {
			continue
		}
		for _, ct := range m.Commands {
			if _, ok := registered[ct]; !ok {
				t.Errorf("mechanic %s references command %s which is not in daggerheartCommandDefinitions", m.ID, ct)
			}
		}
	}
}

func TestManifest_ImplementedEventsAreRegistered(t *testing.T) {
	registered := make(map[event.Type]struct{}, len(daggerheartEventDefinitions))
	for _, def := range daggerheartEventDefinitions {
		registered[def.Type] = struct{}{}
	}

	for _, m := range MechanicsManifest() {
		if m.Status != MechanicImplemented {
			continue
		}
		for _, et := range m.Events {
			if _, ok := registered[et]; !ok {
				t.Errorf("mechanic %s references event %s which is not in daggerheartEventDefinitions", m.ID, et)
			}
		}
	}
}

func TestManifest_AllRegisteredCommandsAreCoveredByManifest(t *testing.T) {
	covered := make(map[command.Type]struct{})
	for _, m := range MechanicsManifest() {
		for _, ct := range m.Commands {
			covered[ct] = struct{}{}
		}
	}
	exempt := map[command.Type]struct{}{
		commandTypeCharacterProfileReplace: {},
		commandTypeCharacterProfileDelete:  {},
	}

	for _, def := range daggerheartCommandDefinitions {
		if _, ok := exempt[def.Type]; ok {
			continue
		}
		if _, ok := covered[def.Type]; !ok {
			t.Errorf("command %s is registered but not referenced by any mechanic in the manifest", def.Type)
		}
	}
}

func TestManifest_ScenarioTagsExist(t *testing.T) {
	// Resolve the scenario directory relative to this test file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// thisFile: .../bridge/daggerheart/mechanics_manifest_test.go
	// scenarios: .../internal/test/game/scenarios/systems/daggerheart/
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "..")
	scenarioDir := filepath.Join(repoRoot, "internal", "test", "game", "scenarios", "systems", "daggerheart")

	// Build set of existing scenario basenames.
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		t.Fatalf("reading scenario directory: %v", err)
	}
	existing := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if ext := filepath.Ext(name); ext == ".lua" {
			existing[name[:len(name)-len(ext)]] = struct{}{}
		}
	}

	for _, m := range MechanicsManifest() {
		for _, tag := range m.ScenarioTags {
			if _, ok := existing[tag]; !ok {
				t.Errorf("mechanic %s references scenario tag %q which does not exist as a .lua file in %s", m.ID, tag, scenarioDir)
			}
		}
	}
}

func TestManifest_DeriveImplementationStage(t *testing.T) {
	stage := DeriveImplementationStage()
	// All Required mechanics are now implemented (including leveling and multiclassing).
	if stage != bridge.ImplementationStageComplete {
		t.Fatalf("DeriveImplementationStage() = %s, want COMPLETE", stage)
	}
}
