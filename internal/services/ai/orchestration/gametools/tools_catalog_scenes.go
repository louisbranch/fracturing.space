package gametools

import "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"

func sceneToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
		{
			Tool: orchestration.Tool{
				Name:        "scene_create",
				Description: "Creates a new scene within the current session and activates it by default unless activate is set to false",
				InputSchema: schemaObject(map[string]schemaProperty{
					"name":          {Type: "string", Description: "scene title"},
					"description":   {Type: "string", Description: "scene framing description"},
					"character_ids": {Type: "array", Description: "optional starting character identifiers", Items: &schemaProperty{Type: "string"}},
					"activate":      {Type: "boolean", Description: "optional explicit activation flag; defaults to true"},
				}),
			},
			Execute: (*DirectSession).sceneCreate,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_update",
				Description: "Updates scene metadata (name or description) for an existing scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":    {Type: "string", Description: "scene identifier"},
					"name":        {Type: "string", Description: "updated scene title"},
					"description": {Type: "string", Description: "updated scene framing description"},
				}),
			},
			Execute: (*DirectSession).sceneUpdate,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_end",
				Description: "Ends a scene, marking it as no longer active",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier"},
					"reason":   {Type: "string", Description: "optional reason for ending the scene"},
				}),
			},
			Execute: (*DirectSession).sceneEnd,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_transition",
				Description: "Atomically ends the current scene and creates a new one, carrying characters forward",
				InputSchema: schemaObject(map[string]schemaProperty{
					"source_scene_id": {Type: "string", Description: "scene to end (defaults to active scene)"},
					"name":            {Type: "string", Description: "new scene title"},
					"description":     {Type: "string", Description: "new scene framing description"},
				}),
			},
			Execute: (*DirectSession).sceneTransition,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_add_character",
				Description: "Adds a character to a scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "scene identifier"},
					"character_id": {Type: "string", Description: "character identifier to add"},
				}),
			},
			Execute: (*DirectSession).sceneAddCharacter,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_remove_character",
				Description: "Removes a character from a scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "scene identifier"},
					"character_id": {Type: "string", Description: "character identifier to remove"},
				}),
			},
			Execute: (*DirectSession).sceneRemoveCharacter,
		},
	}
}
