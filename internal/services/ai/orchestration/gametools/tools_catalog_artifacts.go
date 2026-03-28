package gametools

import "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"

func artifactToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
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
	}
}
