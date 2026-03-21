package gametools

import "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"

// productionTools returns the 23 static tool definitions that match
// mcpbridge.productionToolNames.
func productionTools() []orchestration.Tool {
	return []orchestration.Tool{
		// Artifact tools
		{
			Name:        "campaign_artifact_list",
			Description: "Lists AI GM campaign artifacts such as skills.md, story.md, memory.md, and working notes",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
			}),
		},
		{
			Name:        "campaign_artifact_get",
			Description: "Reads one AI GM campaign artifact such as story.md, memory.md, or a working note",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"path":        {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
			}, "path"),
		},
		{
			Name:        "campaign_artifact_upsert",
			Description: "Writes one mutable AI GM campaign artifact such as story.md, memory.md, or a working note",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"path":        {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
				"content":     {Type: "string", Description: "full replacement markdown content"},
			}, "path", "content"),
		},

		// Scene tools
		{
			Name:        "scene_create",
			Description: "Creates a new scene within the current session; creation alone does not make it the authoritative active scene",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id":   {Type: "string", Description: "campaign identifier (defaults to context)"},
				"session_id":    {Type: "string", Description: "session identifier (defaults to context)"},
				"name":          {Type: "string", Description: "scene title"},
				"description":   {Type: "string", Description: "scene framing description"},
				"character_ids": {Type: "array", Description: "optional starting character identifiers", Items: &schemaProperty{Type: "string"}},
			}, "name"),
		},

		// Interaction tools
		{
			Name:        "interaction_active_scene_set",
			Description: "Sets the authoritative active scene for the current session",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":    {Type: "string", Description: "scene identifier"},
			}, "scene_id"),
		},
		{
			Name:        "interaction_scene_player_phase_start",
			Description: "Starts a new player phase on the active scene from a GM frame",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id":   {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":      {Type: "string", Description: "scene identifier (defaults to active scene)"},
				"frame_text":    {Type: "string", Description: "GM frame text shown to acting players"},
				"character_ids": {Type: "array", Description: "acting character identifiers", Items: &schemaProperty{Type: "string"}},
			}, "frame_text", "character_ids"),
		},
		{
			Name:        "interaction_scene_player_phase_accept",
			Description: "Accepts the active scene player phase after GM review and returns authority to the GM",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":    {Type: "string", Description: "scene identifier (defaults to active scene)"},
			}),
		},
		{
			Name:        "interaction_scene_player_revisions_request",
			Description: "Requests revisions for one or more participant slots in the active scene player phase",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":    {Type: "string", Description: "scene identifier (defaults to active scene)"},
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
			}, "revisions"),
		},
		{
			Name:        "interaction_scene_player_phase_end",
			Description: "Ends the active scene player phase early under GM control",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":    {Type: "string", Description: "scene identifier (defaults to active scene)"},
				"reason":      {Type: "string", Description: "optional GM-supplied reason"},
			}),
		},
		{
			Name:        "interaction_scene_gm_output_commit",
			Description: "Commits authoritative GM narration or instructions for the active scene",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"scene_id":    {Type: "string", Description: "scene identifier (defaults to active scene)"},
				"text":        {Type: "string", Description: "authoritative GM narration or instruction text"},
			}, "text"),
		},
		{
			Name:        "interaction_ooc_pause",
			Description: "Opens the session-level out-of-character pause overlay",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"reason":      {Type: "string", Description: "optional OOC pause reason"},
			}),
		},
		{
			Name:        "interaction_ooc_post",
			Description: "Posts one append-only out-of-character transcript message",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
				"body":        {Type: "string", Description: "out-of-character message body"},
			}, "body"),
		},
		{
			Name:        "interaction_ooc_ready_mark",
			Description: "Marks the caller as ready to resume from the current OOC pause",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
			}),
		},
		{
			Name:        "interaction_ooc_ready_clear",
			Description: "Clears the caller's ready-to-resume state for the current OOC pause",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
			}),
		},
		{
			Name:        "interaction_ooc_resume",
			Description: "Resumes in-character scene play from the current OOC pause",
			InputSchema: schemaObject(map[string]schemaProperty{
				"campaign_id": {Type: "string", Description: "campaign identifier (defaults to context)"},
			}),
		},

		// Daggerheart tools
		{
			Name:        "duality_action_roll",
			Description: "Rolls Duality dice for an action",
			InputSchema: schemaObject(map[string]schemaProperty{
				"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
				"difficulty": {Type: "integer", Description: "optional difficulty target"},
				"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
					"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
					"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
				}},
			}, "modifier"),
		},
		{
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
			}, "dice"),
		},
		{
			Name:        "duality_outcome",
			Description: "Evaluates a duality outcome from known dice",
			InputSchema: schemaObject(map[string]schemaProperty{
				"hope":       {Type: "integer", Description: "hope die result"},
				"fear":       {Type: "integer", Description: "fear die result"},
				"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
				"difficulty": {Type: "integer", Description: "optional difficulty target"},
			}, "hope", "fear", "modifier"),
		},
		{
			Name:        "duality_explain",
			Description: "Explains a duality outcome from known dice",
			InputSchema: schemaObject(map[string]schemaProperty{
				"hope":       {Type: "integer", Description: "hope die result"},
				"fear":       {Type: "integer", Description: "fear die result"},
				"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
				"difficulty": {Type: "integer", Description: "optional difficulty target"},
				"request_id": {Type: "string", Description: "optional correlation identifier"},
			}, "hope", "fear", "modifier"),
		},
		{
			Name:        "duality_probability",
			Description: "Computes outcome probabilities across duality dice",
			InputSchema: schemaObject(map[string]schemaProperty{
				"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
				"difficulty": {Type: "integer", Description: "difficulty target"},
			}, "modifier", "difficulty"),
		},
		{
			Name:        "duality_rules_version",
			Description: "Describes the Duality ruleset semantics",
			InputSchema: schemaObject(nil),
		},

		// Reference tools
		{
			Name:        "system_reference_search",
			Description: "Searches the configured read-only game-system reference corpus",
			InputSchema: schemaObject(map[string]schemaProperty{
				"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
				"query":       {Type: "string", Description: "search query"},
				"max_results": {Type: "integer", Description: "optional max result count"},
			}, "query"),
		},
		{
			Name:        "system_reference_read",
			Description: "Reads one full document from the configured read-only game-system reference corpus",
			InputSchema: schemaObject(map[string]schemaProperty{
				"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
				"document_id": {Type: "string", Description: "document identifier from search results"},
			}, "document_id"),
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

// schemaObject builds a JSON-Schema-like map for an object type with optional required fields.
func schemaObject(properties map[string]schemaProperty, required ...string) map[string]any {
	schema := map[string]any{
		"type": "object",
	}
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
