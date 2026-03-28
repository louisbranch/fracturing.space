package gametools

import (
	"slices"
)

func newProductionToolDefinitions() []productionToolDefinition {
	var definitions []productionToolDefinition
	definitions = append(definitions, artifactToolDefinitions()...)
	definitions = append(definitions, sceneToolDefinitions()...)
	definitions = append(definitions, interactionToolDefinitions()...)
	definitions = append(definitions, daggerheartToolDefinitions()...)
	definitions = append(definitions, diceUtilityToolDefinitions()...)
	definitions = append(definitions, referenceToolDefinitions()...)
	return definitions
}

func interactionSchemaProperty(description string) schemaProperty {
	return schemaProperty{
		Type:        "object",
		Description: description + "; prompt beats ask only for the acting player character's next action, choice, dialogue, or commitment and never outsource NPC dialogue or story outcomes to the player",
		Properties: map[string]schemaProperty{
			"title":         {Type: "string", Description: "short interaction title"},
			"character_ids": {Type: "array", Description: "characters addressed by the interaction", Items: &schemaProperty{Type: "string"}},
			"beats": {
				Type:        "array",
				Description: "ordered beats that make up the GM interaction; keep related prose in one beat even across paragraphs, start a new beat only when the function or information context materially changes, use resolution and consequence only for adjudicated results, and end with a prompt beat when players should act next",
				Items: &schemaProperty{
					Type: "object",
					Properties: map[string]schemaProperty{
						"beat_id": {Type: "string", Description: "optional stable beat identifier"},
						"type":    {Type: "string", Description: "beat type: fiction, prompt, resolution, consequence, or guidance; use resolution only after adjudication and prompt only for player-character handoff"},
						"text":    {Type: "string", Description: "beat body text; may span multiple paragraphs when it serves one coherent beat; prompt text must not ask the player to script NPC dialogue or story outcomes"},
					},
				},
			},
		},
	}
}

// schemaProperty is a minimal JSON Schema property definition.
type schemaProperty struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Items       *schemaProperty           `json:"items,omitempty"`
	Properties  map[string]schemaProperty `json:"properties,omitempty"`
}

// schemaObject builds a JSON-Schema-like map for an object type.
// All properties are marked required for OpenAI strict mode compatibility;
// optional fields accept empty strings and are defaulted server-side.
func schemaObject(properties map[string]schemaProperty, _ ...string) map[string]any {
	schema := map[string]any{
		"type": "object",
	}
	if len(properties) > 0 {
		schema["properties"] = properties
		required := make([]string, 0, len(properties))
		for name := range properties {
			required = append(required, name)
		}
		slices.Sort(required)
		schema["required"] = required
	}
	return schema
}

func rngSchemaProperty(description string) schemaProperty {
	return schemaProperty{
		Type:        "object",
		Description: description,
		Properties: map[string]schemaProperty{
			"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
			"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
		},
	}
}
