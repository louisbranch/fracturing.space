package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParticipantChainingCreatesSteps(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("chain")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Participant + character
scene:participant({name = "John"}):character({name = "Frodo"})

return scene
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 3 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 3)
	}

	participant := scenario.Steps[1]
	if participant.Kind != "participant" {
		t.Fatalf("step kind = %q, want %q", participant.Kind, "participant")
	}
	if participant.Args["name"] != "John" {
		t.Fatalf("participant name = %v, want John", participant.Args["name"])
	}

	character := scenario.Steps[2]
	if character.Kind != "character" {
		t.Fatalf("step kind = %q, want %q", character.Kind, "character")
	}
	if character.Args["name"] != "Frodo" {
		t.Fatalf("character name = %v, want Frodo", character.Args["name"])
	}
	if character.Args["participant"] != "John" {
		t.Fatalf("character participant = %v, want John", character.Args["participant"])
	}
	if character.Args["control"] != "participant" {
		t.Fatalf("character control = %v, want participant", character.Args["control"])
	}
}

func TestParticipantChainingOverridesDefaults(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("chain")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Participant + character overrides
scene:participant({name = "Ada", role = "GM", controller = "AI"}):character({name = "Sam", kind = "NPC", control = "gm"})

return scene
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 3 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 3)
	}

	participant := scenario.Steps[1]
	if participant.Kind != "participant" {
		t.Fatalf("step kind = %q, want %q", participant.Kind, "participant")
	}
	if participant.Args["name"] != "Ada" {
		t.Fatalf("participant name = %v, want Ada", participant.Args["name"])
	}
	if participant.Args["role"] != "GM" {
		t.Fatalf("participant role = %v, want GM", participant.Args["role"])
	}
	if participant.Args["controller"] != "AI" {
		t.Fatalf("participant controller = %v, want AI", participant.Args["controller"])
	}

	character := scenario.Steps[2]
	if character.Kind != "character" {
		t.Fatalf("step kind = %q, want %q", character.Kind, "character")
	}
	if character.Args["name"] != "Sam" {
		t.Fatalf("character name = %v, want Sam", character.Args["name"])
	}
	if character.Args["kind"] != "NPC" {
		t.Fatalf("character kind = %v, want NPC", character.Args["kind"])
	}
	if character.Args["control"] != "gm" {
		t.Fatalf("character control = %v, want gm", character.Args["control"])
	}
}

func TestScenarioParticipantRequiresName(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("missing_participant")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Missing participant name
scene:participant({})

return scene
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "participant name is required") {
		t.Fatalf("error = %q, want participant name is required", err.Error())
	}
}

func TestScenarioParticipantCharacterRequiresName(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("missing_character")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Missing character name
scene:participant({name = "John"}):character({})

return scene
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "character name is required") {
		t.Fatalf("error = %q, want character name is required", err.Error())
	}
}

func TestScenarioSetSpotlightCreatesStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("spotlight")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Force spotlight to a character
scene:set_spotlight({target = "Frodo", expect_spotlight = "Frodo"})

return scene
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}

	step := scenario.Steps[1]
	if step.Kind != "set_spotlight" {
		t.Fatalf("step kind = %q, want %q", step.Kind, "set_spotlight")
	}
	if step.Args["target"] != "Frodo" {
		t.Fatalf("target = %v, want Frodo", step.Args["target"])
	}
	if step.Args["expect_spotlight"] != "Frodo" {
		t.Fatalf("expect_spotlight = %v, want Frodo", step.Args["expect_spotlight"])
	}
}

func TestScenarioClearSpotlightCreatesStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scene = Scenario.new("clear_spotlight")
scene:campaign({name = "Test", system = "DAGGERHEART"})

-- Clear spotlight when needed
scene:clear_spotlight()

return scene
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}

	step := scenario.Steps[1]
	if step.Kind != "clear_spotlight" {
		t.Fatalf("step kind = %q, want %q", step.Kind, "clear_spotlight")
	}
}

func writeScenarioFixture(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.lua")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write scenario: %v", err)
	}
	return path
}
