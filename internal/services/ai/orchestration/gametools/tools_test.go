package gametools

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

func TestProductionToolRegistryMatchesSessionCatalog(t *testing.T) {
	sess := NewDirectSession(Clients{}, SessionContext{})

	tools, err := sess.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			t.Fatalf("tool has empty name: %#v", tool)
		}
		if _, exists := seen[tool.Name]; exists {
			t.Fatalf("duplicate tool name %q", tool.Name)
		}
		seen[tool.Name] = struct{}{}
		names = append(names, tool.Name)
		if _, ok := defaultRegistry.lookup(tool.Name); !ok {
			t.Fatalf("tool %q is listed but not dispatchable", tool.Name)
		}
	}

	if !reflect.DeepEqual(names, ProductionToolNames()) {
		t.Fatalf("catalog names = %#v, want %#v", names, ProductionToolNames())
	}
}

func TestProductionToolDescriptionsUseBeatBasedInteractionGuidance(t *testing.T) {
	sess := NewDirectSession(Clients{}, SessionContext{})

	tools, err := sess.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	byName := make(map[string]orchestration.Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	phaseStart, ok := byName["interaction_open_scene_player_phase"]
	if !ok {
		t.Fatal("missing interaction_open_scene_player_phase tool")
	}
	if !strings.Contains(phaseStart.Description, "structured GM interaction") {
		t.Fatalf("phase-start description = %q", phaseStart.Description)
	}
	if !strings.Contains(phaseStart.Description, "players should act next") {
		t.Fatalf("phase-start handoff guidance = %q", phaseStart.Description)
	}
	if !strings.Contains(promptBeatDescription(phaseStart.InputSchema), "prompt beat") {
		t.Fatalf("phase-start schema missing prompt beat guidance: %#v", phaseStart.InputSchema)
	}
	if !strings.Contains(beatDescription(phaseStart.InputSchema), "keep related prose in one beat even across paragraphs") {
		t.Fatalf("phase-start schema missing beat granularity guidance: %#v", phaseStart.InputSchema)
	}
	if !strings.Contains(promptBeatDescription(phaseStart.InputSchema), "never outsource NPC dialogue") {
		t.Fatalf("phase-start schema missing narrator-authority guidance: %#v", phaseStart.InputSchema)
	}
	if !strings.Contains(beatDescription(phaseStart.InputSchema), "use resolution and consequence only for adjudicated results") {
		t.Fatalf("phase-start schema missing adjudication beat guidance: %#v", phaseStart.InputSchema)
	}
	if !strings.Contains(beatTextDescription(phaseStart.InputSchema), "may span multiple paragraphs") {
		t.Fatalf("phase-start schema missing beat text paragraph guidance: %#v", phaseStart.InputSchema)
	}
	if interactionSchemaHasProperty(phaseStart.InputSchema, "interaction", "illustration") {
		t.Fatalf("phase-start schema unexpectedly exposes illustration: %#v", phaseStart.InputSchema)
	}

	commit, ok := byName["interaction_record_scene_gm_interaction"]
	if !ok {
		t.Fatal("missing interaction_record_scene_gm_interaction tool")
	}
	if !strings.Contains(commit.Description, "without opening a player phase") {
		t.Fatalf("commit description = %q", commit.Description)
	}
	if interactionSchemaHasProperty(commit.InputSchema, "interaction", "illustration") {
		t.Fatalf("commit schema unexpectedly exposes illustration: %#v", commit.InputSchema)
	}

	interrupt, ok := byName["interaction_session_ooc_resolve"]
	if !ok {
		t.Fatal("missing interaction_session_ooc_resolve tool")
	}
	if !strings.Contains(interrupt.Description, "newly opened player phase") {
		t.Fatalf("interrupt description = %q", interrupt.Description)
	}
	if !strings.Contains(promptBeatDescription(interrupt.InputSchema), "prompt beat") {
		t.Fatalf("interrupt schema missing prompt beat guidance: %#v", interrupt.InputSchema)
	}
	if interactionSchemaHasProperty(interrupt.InputSchema, "open_player_phase", "interaction", "illustration") {
		t.Fatalf("interrupt schema unexpectedly exposes illustration: %#v", interrupt.InputSchema)
	}

	sheet, ok := byName["character_sheet_read"]
	if !ok {
		t.Fatal("missing character_sheet_read tool")
	}
	if !strings.Contains(sheet.Description, "traits") || !strings.Contains(sheet.Description, "domain cards") {
		t.Fatalf("character_sheet_read description = %q", sheet.Description)
	}

	interactionState, ok := byName["interaction_state_read"]
	if !ok {
		t.Fatal("missing interaction_state_read tool")
	}
	if !strings.Contains(interactionState.Description, "diagnose") || !strings.Contains(interactionState.Description, "active scene") {
		t.Fatalf("interaction_state_read description = %q", interactionState.Description)
	}

	board, ok := byName["daggerheart_combat_board_read"]
	if !ok {
		t.Fatal("missing daggerheart_combat_board_read tool")
	}
	if !strings.Contains(board.Description, "GM Fear") || !strings.Contains(board.Description, "countdowns") || !strings.Contains(board.Description, "adversaries") || !strings.Contains(board.Description, "diagnostic") {
		t.Fatalf("daggerheart_combat_board_read description = %q", board.Description)
	}

	actionResolve, ok := byName["daggerheart_action_roll_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_action_roll_resolve tool")
	}
	if !strings.Contains(actionResolve.Description, "authoritative Daggerheart action roll") || !strings.Contains(actionResolve.Description, "applies its outcome") {
		t.Fatalf("daggerheart_action_roll_resolve description = %q", actionResolve.Description)
	}

	gmMove, ok := byName["daggerheart_gm_move_apply"]
	if !ok {
		t.Fatal("missing daggerheart_gm_move_apply tool")
	}
	if !strings.Contains(gmMove.Description, "Spends Fear") || !strings.Contains(gmMove.Description, "authoritative Daggerheart GM move") || !strings.Contains(gmMove.Description, "exactly one spend target") {
		t.Fatalf("daggerheart_gm_move_apply description = %q", gmMove.Description)
	}

	adversaryCreate, ok := byName["daggerheart_adversary_create"]
	if !ok {
		t.Fatal("missing daggerheart_adversary_create tool")
	}
	if !strings.Contains(adversaryCreate.Description, "Creates one Daggerheart adversary") || !strings.Contains(adversaryCreate.Description, "current session scene") {
		t.Fatalf("daggerheart_adversary_create description = %q", adversaryCreate.Description)
	}

	countdownCreate, ok := byName["daggerheart_scene_countdown_create"]
	if !ok {
		t.Fatal("missing daggerheart_scene_countdown_create tool")
	}
	if !strings.Contains(countdownCreate.Description, "Creates one Daggerheart scene countdown") || !strings.Contains(countdownCreate.Description, "current session scene") || !strings.Contains(countdownCreate.Description, "fixed_starting_value") {
		t.Fatalf("daggerheart_scene_countdown_create description = %q", countdownCreate.Description)
	}

	countdownUpdate, ok := byName["daggerheart_scene_countdown_advance"]
	if !ok {
		t.Fatal("missing daggerheart_scene_countdown_advance tool")
	}
	if !strings.Contains(countdownUpdate.Description, "Advances one Daggerheart scene countdown") || !strings.Contains(countdownUpdate.Description, "positive amount") {
		t.Fatalf("daggerheart_scene_countdown_advance description = %q", countdownUpdate.Description)
	}

	adversaryUpdate, ok := byName["daggerheart_adversary_update"]
	if !ok {
		t.Fatal("missing daggerheart_adversary_update tool")
	}
	if !strings.Contains(adversaryUpdate.Description, "Updates one Daggerheart adversary") || !strings.Contains(adversaryUpdate.Description, "current scene board") {
		t.Fatalf("daggerheart_adversary_update description = %q", adversaryUpdate.Description)
	}

	attackFlow, ok := byName["daggerheart_attack_flow_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_attack_flow_resolve tool")
	}
	if !strings.Contains(attackFlow.Description, "authoritative Daggerheart attack flow") || !strings.Contains(attackFlow.Description, "damage application") || !strings.Contains(attackFlow.Description, "default attack profile") {
		t.Fatalf("daggerheart_attack_flow_resolve description = %q", attackFlow.Description)
	}

	adversaryAttackFlow, ok := byName["daggerheart_adversary_attack_flow_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_adversary_attack_flow_resolve tool")
	}
	if !strings.Contains(adversaryAttackFlow.Description, "authoritative Daggerheart adversary attack flow") || !strings.Contains(adversaryAttackFlow.Description, "damage application") {
		t.Fatalf("daggerheart_adversary_attack_flow_resolve description = %q", adversaryAttackFlow.Description)
	}

	groupActionFlow, ok := byName["daggerheart_group_action_flow_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_group_action_flow_resolve tool")
	}
	if !strings.Contains(groupActionFlow.Description, "authoritative Daggerheart group action flow") || !strings.Contains(groupActionFlow.Description, "leader roll") {
		t.Fatalf("daggerheart_group_action_flow_resolve description = %q", groupActionFlow.Description)
	}

	reactionFlow, ok := byName["daggerheart_reaction_flow_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_reaction_flow_resolve tool")
	}
	if !strings.Contains(reactionFlow.Description, "authoritative Daggerheart reaction flow") || !strings.Contains(reactionFlow.Description, "reaction outcome") {
		t.Fatalf("daggerheart_reaction_flow_resolve description = %q", reactionFlow.Description)
	}

	tagTeamFlow, ok := byName["daggerheart_tag_team_flow_resolve"]
	if !ok {
		t.Fatal("missing daggerheart_tag_team_flow_resolve tool")
	}
	if !strings.Contains(tagTeamFlow.Description, "authoritative Daggerheart tag-team flow") || !strings.Contains(tagTeamFlow.Description, "selected combined outcome") {
		t.Fatalf("daggerheart_tag_team_flow_resolve description = %q", tagTeamFlow.Description)
	}

	referenceSearch, ok := byName["system_reference_search"]
	if !ok {
		t.Fatal("missing system_reference_search tool")
	}
	if !strings.Contains(referenceSearch.Description, "exact wording") || !strings.Contains(referenceSearch.Description, "procedure choice is unclear") {
		t.Fatalf("system_reference_search description = %q", referenceSearch.Description)
	}

	referenceRead, ok := byName["system_reference_read"]
	if !ok {
		t.Fatal("missing system_reference_read tool")
	}
	if !strings.Contains(referenceRead.Description, "search result still needs exact wording") {
		t.Fatalf("system_reference_read description = %q", referenceRead.Description)
	}
}

func promptBeatDescription(schema any) string {
	description, _ := interactionSchemaDescriptions(schema)
	return description
}

func beatDescription(schema any) string {
	_, beats := interactionSchemaDescriptions(schema)
	return beats
}

func beatTextDescription(schema any) string {
	root, ok := schema.(map[string]any)
	if !ok {
		return ""
	}
	properties, ok := root["properties"].(map[string]schemaProperty)
	if !ok {
		return ""
	}
	if interaction, ok := properties["interaction"]; ok {
		return interaction.Properties["beats"].Items.Properties["text"].Description
	}
	replace, ok := properties["open_player_phase"]
	if !ok {
		return ""
	}
	interaction, ok := replace.Properties["interaction"]
	if !ok {
		return ""
	}
	return interaction.Properties["beats"].Items.Properties["text"].Description
}

func interactionSchemaDescriptions(schema any) (string, string) {
	root, ok := schema.(map[string]any)
	if !ok {
		return "", ""
	}
	properties, ok := root["properties"].(map[string]schemaProperty)
	if !ok {
		return "", ""
	}
	if interaction, ok := properties["interaction"]; ok {
		return interaction.Description, interaction.Properties["beats"].Description
	}
	replace, ok := properties["open_player_phase"]
	if !ok {
		return "", ""
	}
	interaction, ok := replace.Properties["interaction"]
	if !ok {
		return "", ""
	}
	return interaction.Description, interaction.Properties["beats"].Description
}

func interactionSchemaHasProperty(schema any, path ...string) bool {
	root, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	properties, ok := root["properties"].(map[string]schemaProperty)
	if !ok {
		return false
	}
	current := properties
	for idx, name := range path {
		prop, ok := current[name]
		if !ok {
			return false
		}
		if idx == len(path)-1 {
			return true
		}
		current = prop.Properties
	}
	return false
}
