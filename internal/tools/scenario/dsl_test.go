package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioSupportsArbitraryRootAlias(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local game = Scenario.new("alias")
game:campaign({name = "Test", system = "DAGGERHEART"})

-- Add one character through the alias.
game:pc("Frodo")

return game
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if scenario.Name != "alias" {
		t.Fatalf("scenario name = %q, want %q", scenario.Name, "alias")
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}
	if scenario.Steps[1].Kind != "character" {
		t.Fatalf("step kind = %q, want %q", scenario.Steps[1].Kind, "character")
	}
}

func TestParticipantChainingCreatesSteps(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("chain")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Participant + character
scn:participant({name = "John"}):character({name = "Frodo"})

return scn
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
local scn = Scenario.new("chain")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Participant + character overrides
scn:participant({name = "Ada", role = "GM", controller = "AI"}):character({name = "Sam", kind = "NPC", control = "gm"})

return scn
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
local scn = Scenario.new("missing_participant")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Missing participant name
scn:participant({})

return scn
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
local scn = Scenario.new("missing_character")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Missing character name
scn:participant({name = "John"}):character({})

return scn
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
local scn = Scenario.new("spotlight")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Force spotlight to a character
scn:set_spotlight({target = "Frodo", expect_spotlight = "Frodo"})

return scn
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
local scn = Scenario.new("clear_spotlight")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Clear spotlight when needed
scn:clear_spotlight()

return scn
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

func TestScenarioAdversaryReactionCreatesStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("adversary_reaction")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Trigger a reactive adversary effect and cooldown toggle.
dh:adversary_reaction({actor = "Saruman", target = "Frodo", damage = 7, damage_type = "magic", cooldown_note = "warding_sphere:cooldown"})

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}

	step := scenario.Steps[1]
	if step.Kind != "adversary_reaction" {
		t.Fatalf("step kind = %q, want %q", step.Kind, "adversary_reaction")
	}
	if step.System != "DAGGERHEART" {
		t.Fatalf("step system = %q, want DAGGERHEART", step.System)
	}
	if step.Args["actor"] != "Saruman" {
		t.Fatalf("actor = %v, want Saruman", step.Args["actor"])
	}
	if step.Args["target"] != "Frodo" {
		t.Fatalf("target = %v, want Frodo", step.Args["target"])
	}
}

func TestScenarioGroupReactionCreatesStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("group_reaction")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Roll reactions for multiple targets and apply failure-only effects.
dh:group_reaction({targets = {"Frodo", "Sam"}, trait = "agility", difficulty = 15, failure_conditions = {"VULNERABLE"}, source = "snowblind_trap"})

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}

	step := scenario.Steps[1]
	if step.Kind != "group_reaction" {
		t.Fatalf("step kind = %q, want %q", step.Kind, "group_reaction")
	}
	if step.System != "DAGGERHEART" {
		t.Fatalf("step system = %q, want DAGGERHEART", step.System)
	}
}

func TestScenarioSystemScopedMethodsAreNotAvailableOnScene(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("legacy_scene_method")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Legacy style is intentionally removed.
scn:attack({actor = "Frodo", target = "Nazgul"})

return scn
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires a system handle") {
		t.Fatalf("error = %q, want migration guidance", err.Error())
	}
	if !strings.Contains(err.Error(), "<SYSTEM_ID>") {
		t.Fatalf("error = %q, want generic system placeholder", err.Error())
	}
}

func TestScenarioSystemRequiresRegisteredSystem(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("unknown_system")
local dh = scn:system("UNKNOWN")
scn:campaign({name = "Test", system = "DAGGERHEART"})
dh:gm_fear(1)

return scn
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported system") && !strings.Contains(err.Error(), "unsupported scenario system") {
		t.Fatalf("error = %q, want unsupported system", err.Error())
	}
}

func TestValidateScenarioCommentsRequiresCommentForSceneBlock(t *testing.T) {
	path := writeScenarioFixture(t, `local scn = Scenario.new("no-comment")
scn:campaign({name = "Test", system = "DAGGERHEART"})

return scn
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected validation error for missing block comment")
	}
	if !strings.Contains(err.Error(), "scenario block missing comment") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "scenario block missing comment")
	}
}

func TestValidateScenarioCommentsAllowsCommentedSceneBlock(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("commented")
scn:campaign({name = "Test", system = "DAGGERHEART"})

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if scenario.Name != "commented" {
		t.Fatalf("scenario name = %q, want %q", scenario.Name, "commented")
	}
}

func TestValidateScenarioCommentsRequiresCommentForSystemHandleBlock(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("system-no-comment")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

dh:gm_fear(1)

return scn
`)

	_, err := LoadScenarioFromFile(path)
	if err == nil {
		t.Fatal("expected validation error for missing system block comment")
	}
	if !strings.Contains(err.Error(), "scenario block missing comment") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "scenario block missing comment")
	}
}

func TestValidateScenarioCommentsAllowsCommentedSystemHandleBlock(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("system-commented")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Increase GM fear to force follow-up branch behavior.
dh:gm_fear(1)

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if scenario.Name != "system-commented" {
		t.Fatalf("scenario name = %q, want %q", scenario.Name, "system-commented")
	}
	if len(scenario.Steps) != 2 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 2)
	}
	if scenario.Steps[1].Kind != "gm_fear" {
		t.Fatalf("step kind = %q, want gm_fear", scenario.Steps[1].Kind)
	}
	if scenario.Steps[1].System != "DAGGERHEART" {
		t.Fatalf("step system = %q, want DAGGERHEART", scenario.Steps[1].System)
	}
}

func TestLoadScenarioFromFileWithoutCommentValidationAllowsMissingComment(t *testing.T) {
	path := writeScenarioFixture(t, `local scn = Scenario.new("no-comment")
scn:campaign({name = "Test", system = "DAGGERHEART"})

return scn
`)

	_, err := LoadScenarioFromFileWithOptions(path, false)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
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
