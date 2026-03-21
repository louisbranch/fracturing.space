package gametools

import (
	"context"
	"fmt"
	"slices"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type toolExecutor func(*DirectSession, context.Context, []byte) (orchestration.ToolResult, error)

type productionToolDefinition struct {
	Tool    orchestration.Tool
	Execute toolExecutor
}

type productionToolRegistry struct {
	definitions []productionToolDefinition
	byName      map[string]productionToolDefinition
}

// defaultRegistry is the standard production registry used by NewDirectDialer.
// Kept as a package-level cache since the definitions are static; injection
// happens at the DirectDialer/DirectSession level so tests can substitute.
var defaultRegistry = newProductionToolRegistry()

func newProductionToolRegistry() productionToolRegistry {
	definitions := []productionToolDefinition{
		// Artifact tools
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_list",
				Description: "Lists AI GM campaign artifacts such as skills.md, story.md, memory.md, and working notes",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).artifactList,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_get",
				Description: "Reads one AI GM campaign artifact such as story.md, memory.md, or a working note",
				InputSchema: schemaObject(map[string]schemaProperty{
					"path": {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
				}),
			},
			Execute: (*DirectSession).artifactGet,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_upsert",
				Description: "Writes one mutable AI GM campaign artifact such as story.md, memory.md, or a working note",
				InputSchema: schemaObject(map[string]schemaProperty{
					"path":    {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
					"content": {Type: "string", Description: "full replacement markdown content"},
				}),
			},
			Execute: (*DirectSession).artifactUpsert,
		},

		// Memory section tools
		{
			Tool: orchestration.Tool{
				Name:        "campaign_memory_section_read",
				Description: "Reads one ## heading section from memory.md without fetching the full document",
				InputSchema: schemaObject(map[string]schemaProperty{
					"heading": {Type: "string", Description: "section heading to read, e.g. NPCs, Plot Hooks, World State"},
				}),
			},
			Execute: (*DirectSession).memorySectionRead,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_memory_section_update",
				Description: "Replaces or appends one ## heading section in memory.md without disturbing other sections",
				InputSchema: schemaObject(map[string]schemaProperty{
					"heading": {Type: "string", Description: "section heading to update or create, e.g. NPCs, Plot Hooks, World State"},
					"content": {Type: "string", Description: "new section body content (replaces existing body for this heading)"},
				}),
			},
			Execute: (*DirectSession).memorySectionUpdate,
		},

		// Scene tools
		{
			Tool: orchestration.Tool{
				Name:        "scene_create",
				Description: "Creates a new scene within the current session; creation alone does not make it the authoritative active scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"name":          {Type: "string", Description: "scene title"},
					"description":   {Type: "string", Description: "scene framing description"},
					"character_ids": {Type: "array", Description: "optional starting character identifiers", Items: &schemaProperty{Type: "string"}},
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

		// Interaction tools
		{
			Tool: orchestration.Tool{
				Name:        "interaction_active_scene_set",
				Description: "Sets the authoritative active scene for the current session",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier"},
				}),
			},
			Execute: (*DirectSession).interactionSetActiveScene,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_scene_player_phase_start",
				Description: "Starts a new player phase on the active scene from a GM frame",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":      {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"frame_text":    {Type: "string", Description: "GM frame text shown to acting players"},
					"character_ids": {Type: "array", Description: "acting character identifiers", Items: &schemaProperty{Type: "string"}},
				}),
			},
			Execute: (*DirectSession).interactionStartScenePlayerPhase,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_scene_player_phase_accept",
				Description: "Accepts the active scene player phase after GM review and returns authority to the GM",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier (defaults to active scene)"},
				}),
			},
			Execute: (*DirectSession).interactionAcceptScenePlayerPhase,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_scene_player_revisions_request",
				Description: "Requests revisions for one or more participant slots in the active scene player phase",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"revisions": {
						Type:        "array",
						Description: "participant-scoped revision requests",
						Items: &schemaProperty{
							Type: "object",
							Properties: map[string]schemaProperty{
								"participant_id": {Type: "string", Description: "participant identifier that must revise their slot"},
								"reason":         {Type: "string", Description: "GM review reason shown to the participant"},
								"character_ids":  {Type: "array", Description: "optional character identifiers affected by the review request", Items: &schemaProperty{Type: "string"}},
							},
						},
					},
				}),
			},
			Execute: (*DirectSession).interactionRequestScenePlayerRevisions,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_scene_player_phase_end",
				Description: "Ends the active scene player phase early under GM control",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"reason":   {Type: "string", Description: "optional GM-supplied reason"},
				}),
			},
			Execute: (*DirectSession).interactionEndScenePlayerPhase,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_scene_gm_output_commit",
				Description: "Commits authoritative GM narration or instructions for the active scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"text":     {Type: "string", Description: "authoritative GM narration or instruction text"},
				}),
			},
			Execute: (*DirectSession).interactionCommitSceneGMOutput,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_ooc_pause",
				Description: "Opens the session-level out-of-character pause overlay",
				InputSchema: schemaObject(map[string]schemaProperty{
					"reason": {Type: "string", Description: "optional OOC pause reason"},
				}),
			},
			Execute: (*DirectSession).interactionPauseOOC,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_ooc_post",
				Description: "Posts one append-only out-of-character transcript message",
				InputSchema: schemaObject(map[string]schemaProperty{
					"body": {Type: "string", Description: "out-of-character message body"},
				}),
			},
			Execute: (*DirectSession).interactionPostOOC,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_ooc_ready_mark",
				Description: "Marks the caller as ready to resume from the current OOC pause",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionMarkOOCReady,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_ooc_ready_clear",
				Description: "Clears the caller's ready-to-resume state for the current OOC pause",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionClearOOCReady,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_ooc_resume",
				Description: "Resumes in-character scene play from the current OOC pause",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionResumeOOC,
		},

		// Daggerheart tools
		{
			Tool: orchestration.Tool{
				Name:        "duality_action_roll",
				Description: "Rolls Duality dice for an action",
				InputSchema: schemaObject(map[string]schemaProperty{
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
					"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
						"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
						"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
					}},
				}),
			},
			Execute: (*DirectSession).dualityActionRoll,
		},
		{
			Tool: orchestration.Tool{
				Name:        "roll_dice",
				Description: "Rolls arbitrary dice pools",
				InputSchema: schemaObject(map[string]schemaProperty{
					"dice": {Type: "array", Description: "dice specifications to roll", Items: &schemaProperty{
						Type: "object", Properties: map[string]schemaProperty{
							"sides": {Type: "integer", Description: "number of sides for the die"},
							"count": {Type: "integer", Description: "number of dice to roll"},
						},
					}},
					"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
						"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
						"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
					}},
				}),
			},
			Execute: (*DirectSession).rollDice,
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_outcome",
				Description: "Evaluates a duality outcome from known dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"hope":       {Type: "integer", Description: "hope die result"},
					"fear":       {Type: "integer", Description: "fear die result"},
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
				}),
			},
			Execute: (*DirectSession).dualityOutcome,
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_explain",
				Description: "Explains a duality outcome from known dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"hope":       {Type: "integer", Description: "hope die result"},
					"fear":       {Type: "integer", Description: "fear die result"},
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
					"request_id": {Type: "string", Description: "optional correlation identifier"},
				}),
			},
			Execute: (*DirectSession).dualityExplain,
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_probability",
				Description: "Computes outcome probabilities across duality dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "difficulty target"},
				}),
			},
			Execute: (*DirectSession).dualityProbability,
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_rules_version",
				Description: "Describes the Duality ruleset semantics",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).dualityRulesVersion,
		},

		// Reference tools
		{
			Tool: orchestration.Tool{
				Name:        "system_reference_search",
				Description: "Searches the configured read-only game-system reference corpus",
				InputSchema: schemaObject(map[string]schemaProperty{
					"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
					"query":       {Type: "string", Description: "search query"},
					"max_results": {Type: "integer", Description: "optional max result count"},
				}),
			},
			Execute: (*DirectSession).referenceSearch,
		},
		{
			Tool: orchestration.Tool{
				Name:        "system_reference_read",
				Description: "Reads one full document from the configured read-only game-system reference corpus",
				InputSchema: schemaObject(map[string]schemaProperty{
					"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
					"document_id": {Type: "string", Description: "document identifier from search results"},
				}),
			},
			Execute: (*DirectSession).referenceRead,
		},
	}

	byName := make(map[string]productionToolDefinition, len(definitions))
	for i, definition := range definitions {
		name := definition.Tool.Name
		if name == "" {
			panic("gametools: production tool name is required")
		}
		if definition.Execute == nil {
			panic(fmt.Sprintf("gametools: production tool %q is missing an executor", name))
		}
		definitions[i] = definition
		if _, exists := byName[name]; exists {
			panic(fmt.Sprintf("gametools: duplicate production tool %q", name))
		}
		byName[name] = definition
	}

	return productionToolRegistry{
		definitions: definitions,
		byName:      byName,
	}
}

func (r productionToolRegistry) tools() []orchestration.Tool {
	tools := make([]orchestration.Tool, 0, len(r.definitions))
	for _, definition := range r.definitions {
		tools = append(tools, definition.Tool)
	}
	return tools
}

func (r productionToolRegistry) lookup(name string) (productionToolDefinition, bool) {
	definition, ok := r.byName[name]
	return definition, ok
}

// ProductionToolNames returns the concrete production tool profile owned by
// the direct game-tools bridge.
func ProductionToolNames() []string {
	names := make([]string, 0, len(defaultRegistry.definitions))
	for _, definition := range defaultRegistry.definitions {
		names = append(names, definition.Tool.Name)
	}
	return names
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
