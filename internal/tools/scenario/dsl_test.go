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

func TestScenarioGmSpendFearTypedTargetsCreateSteps(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("gm_fear_targets")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Typed GM fear spend targets
dh:gm_spend_fear(1):move("reveal_danger", { description = "Danger closes in." })
dh:gm_spend_fear(1):adversary_spotlight("Shadow Hound")
dh:gm_spend_fear(1):adversary_feature("adversary.shadow-hound", "feature.shadow-hound-pounce")
dh:gm_spend_fear(2):environment_feature("environment.crumbling-bridge", "feature.crumbling-bridge-falling-stones")
dh:gm_spend_fear(1):adversary_experience("adversary.shadow-hound", "Pack Hunter")

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 6 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 6)
	}

	direct := scenario.Steps[1]
	if direct.Kind != "gm_spend_fear" {
		t.Fatalf("step[1].Kind = %q", direct.Kind)
	}
	if direct.Args["spend_target"] != "direct_move" {
		t.Fatalf("direct spend_target = %v, want direct_move", direct.Args["spend_target"])
	}
	if direct.Args["move"] != "reveal_danger" {
		t.Fatalf("direct move = %v, want reveal_danger", direct.Args["move"])
	}

	adversarySpotlight := scenario.Steps[2]
	if adversarySpotlight.Args["spend_target"] != "direct_move" {
		t.Fatalf("adversary spotlight spend_target = %v", adversarySpotlight.Args["spend_target"])
	}
	if adversarySpotlight.Args["move"] != "spotlight" {
		t.Fatalf("adversary spotlight move = %v, want spotlight", adversarySpotlight.Args["move"])
	}
	if adversarySpotlight.Args["target"] != "Shadow Hound" {
		t.Fatalf("adversary spotlight target = %v, want Shadow Hound", adversarySpotlight.Args["target"])
	}

	adversaryFeature := scenario.Steps[3]
	if adversaryFeature.Args["spend_target"] != "adversary_feature" {
		t.Fatalf("adversary feature spend_target = %v", adversaryFeature.Args["spend_target"])
	}
	if adversaryFeature.Args["target"] != "adversary.shadow-hound" {
		t.Fatalf("target = %v", adversaryFeature.Args["target"])
	}
	if adversaryFeature.Args["feature_id"] != "feature.shadow-hound-pounce" {
		t.Fatalf("feature_id = %v", adversaryFeature.Args["feature_id"])
	}

	environmentFeature := scenario.Steps[4]
	if environmentFeature.Args["spend_target"] != "environment_feature" {
		t.Fatalf("environment feature spend_target = %v", environmentFeature.Args["spend_target"])
	}
	if environmentFeature.Args["environment_id"] != "environment.crumbling-bridge" {
		t.Fatalf("environment_id = %v", environmentFeature.Args["environment_id"])
	}
	if environmentFeature.Args["feature_id"] != "feature.crumbling-bridge-falling-stones" {
		t.Fatalf("feature_id = %v", environmentFeature.Args["feature_id"])
	}

	adversaryExperience := scenario.Steps[5]
	if adversaryExperience.Args["spend_target"] != "adversary_experience" {
		t.Fatalf("adversary experience spend_target = %v", adversaryExperience.Args["spend_target"])
	}
	if adversaryExperience.Args["target"] != "adversary.shadow-hound" {
		t.Fatalf("target = %v", adversaryExperience.Args["target"])
	}
	if adversaryExperience.Args["experience_name"] != "Pack Hunter" {
		t.Fatalf("experience_name = %v", adversaryExperience.Args["experience_name"])
	}
}

func TestScenarioInteractionMethodsCreateSteps(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("interaction")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Hand GM authority to a named participant and open a player phase.
scn:interaction_set_gm_authority({participant = "Guide", as = "Guide"})
scn:interaction_start_player_phase({
  scene = "The Bridge",
  interaction = {
    title = "Opening Beat",
    beats = {{type = "prompt", text = "What do you do?"}},
  },
  characters = {"Aria", "Corin"},
  as = "Guide",
})
scn:interaction_post({summary = "Aria rushes forward.", characters = {"Aria"}, as = "Rhea", yield = true})
scn:interaction_resume_ooc()

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 5 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 5)
	}
	if scenario.Steps[1].Kind != "interaction_set_gm_authority" {
		t.Fatalf("step[1].Kind = %q", scenario.Steps[1].Kind)
	}
	if scenario.Steps[1].Args["participant"] != "Guide" {
		t.Fatalf("participant = %v, want Guide", scenario.Steps[1].Args["participant"])
	}
	if scenario.Steps[2].Kind != "interaction_start_player_phase" {
		t.Fatalf("step[2].Kind = %q", scenario.Steps[2].Kind)
	}
	interactionArgs, ok := scenario.Steps[2].Args["interaction"].(map[string]any)
	if !ok {
		t.Fatalf("interaction = %#v, want table", scenario.Steps[2].Args["interaction"])
	}
	beats, ok := interactionArgs["beats"].([]any)
	if !ok || len(beats) != 1 {
		t.Fatalf("interaction.beats = %#v, want single beat", interactionArgs["beats"])
	}
	beat, ok := beats[0].(map[string]any)
	if !ok || beat["text"] != "What do you do?" {
		t.Fatalf("interaction beat = %#v, want prompt", beats[0])
	}
	if scenario.Steps[3].Kind != "interaction_post" {
		t.Fatalf("step[3].Kind = %q", scenario.Steps[3].Kind)
	}
	if scenario.Steps[3].Args["as"] != "Rhea" {
		t.Fatalf("as = %v, want Rhea", scenario.Steps[3].Args["as"])
	}
	if scenario.Steps[4].Kind != "interaction_resume_ooc" {
		t.Fatalf("step[4].Kind = %q", scenario.Steps[4].Kind)
	}
}

func TestScenarioAllInteractionMethodsCreateSteps(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("interaction_all")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Exercise every remaining interaction wrapper once.
scn:interaction_set_active_scene({scene = "The Bridge"})
scn:interaction_yield({as = "Rhea"})
scn:interaction_unyield({as = "Rhea"})
scn:interaction_end_player_phase({reason = "gm_interrupted"})
scn:interaction_resolve_review({
  as = "Guide",
  return_to_gm = true,
  interaction = {title = "Resolution", beats = {{type = "resolution", text = "The bridge settles."}}},
})
scn:interaction_resolve_review({
  as = "Guide",
  interaction = {title = "Clarify", beats = {{type = "guidance", text = "Clarify"}}},
  revisions = {{participant = "Rhea", reason = "Clarify", characters = {"Aria"}}},
})
scn:interaction_resolve_review({
  as = "Guide",
  interaction = {
    title = "Bridge Buckles",
    beats = {
      {type = "fiction", text = "The bridge buckles."},
      {type = "prompt", text = "Who catches the lantern?"},
    },
  },
  characters = {"Aria"},
})
scn:interaction_pause_ooc({reason = "clarify the ruling"})
scn:interaction_post_ooc({as = "Rhea", body = "Question?"})
scn:interaction_ready_ooc({as = "Rhea"})
scn:interaction_clear_ready_ooc({as = "Rhea"})
scn:interaction_resolve_interrupted_phase({as = "Guide", resume_original_phase = true})
scn:interaction_expect({phase_status = "GM_REVIEW", slots = {}, ooc_posts = {}})

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}

	wantKinds := []string{
		"campaign",
		"interaction_set_active_scene",
		"interaction_yield",
		"interaction_unyield",
		"interaction_end_player_phase",
		"interaction_resolve_review",
		"interaction_resolve_review",
		"interaction_resolve_review",
		"interaction_pause_ooc",
		"interaction_post_ooc",
		"interaction_ready_ooc",
		"interaction_clear_ready_ooc",
		"interaction_resolve_interrupted_phase",
		"interaction_expect",
	}
	if len(scenario.Steps) != len(wantKinds) {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), len(wantKinds))
	}
	for index, wantKind := range wantKinds {
		if scenario.Steps[index].Kind != wantKind {
			t.Fatalf("step[%d].Kind = %q, want %q", index, scenario.Steps[index].Kind, wantKind)
		}
	}
	if scenario.Steps[len(scenario.Steps)-1].Args["phase_status"] != "GM_REVIEW" {
		t.Fatalf("phase_status = %v, want GM_REVIEW", scenario.Steps[len(scenario.Steps)-1].Args["phase_status"])
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

func TestScenarioSystemScopedMethodsAreNotAvailableOnScenarioRoot(t *testing.T) {
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

func TestValidateScenarioCommentsRequiresCommentForScenarioBlock(t *testing.T) {
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

func TestValidateScenarioCommentsAllowsCommentedScenarioBlock(t *testing.T) {
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

func TestScenarioCreationWorkflowCreatesSystemStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("creation_workflow")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})
scn:pc("Frodo", { skip_system_readiness = true })

-- Apply explicit Daggerheart creation workflow data.
dh:creation_workflow({
  target = "Frodo",
  class_id = "class.ranger",
  subclass_id = "subclass.beastbound",
  heritage = {
    first_feature_ancestry_id = "heritage.dwarf",
    second_feature_ancestry_id = "heritage.elf",
    ancestry_label = "Stoneleaf",
    community_id = "heritage.highborne"
  }
})

return scn
`)

	scenario, err := LoadScenarioFromFile(path)
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}
	if len(scenario.Steps) != 3 {
		t.Fatalf("steps = %d, want %d", len(scenario.Steps), 3)
	}
	step := scenario.Steps[2]
	if step.Kind != "creation_workflow" {
		t.Fatalf("step kind = %q, want creation_workflow", step.Kind)
	}
	if step.System != "DAGGERHEART" {
		t.Fatalf("step system = %q, want DAGGERHEART", step.System)
	}
	heritage, ok := step.Args["heritage"].(map[string]any)
	if !ok {
		t.Fatalf("heritage = %#v, want map", step.Args["heritage"])
	}
	if heritage["first_feature_ancestry_id"] != "heritage.dwarf" {
		t.Fatalf("first_feature_ancestry_id = %v, want heritage.dwarf", heritage["first_feature_ancestry_id"])
	}
}

func TestScenarioExpectGMFearCreatesSystemStep(t *testing.T) {
	path := writeScenarioFixture(t, `-- Setup
local scn = Scenario.new("expect_gm_fear")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Test", system = "DAGGERHEART"})

-- Assert the current fear pool without mutating it.
dh:expect_gm_fear(2)

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
	if step.Kind != "expect_gm_fear" {
		t.Fatalf("step kind = %q, want expect_gm_fear", step.Kind)
	}
	if step.Args["value"] != 2 {
		t.Fatalf("value = %v, want 2", step.Args["value"])
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
