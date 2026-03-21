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
