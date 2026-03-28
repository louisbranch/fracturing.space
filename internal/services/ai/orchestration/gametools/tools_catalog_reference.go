package gametools

import "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"

func referenceToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
		{
			Tool: orchestration.Tool{
				Name:        "system_reference_search",
				Description: "Searches the configured read-only game-system reference corpus when exact wording or procedure choice is unclear",
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
				Description: "Reads one full document from the configured read-only game-system reference corpus after a search result still needs exact wording",
				InputSchema: schemaObject(map[string]schemaProperty{
					"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
					"document_id": {Type: "string", Description: "document identifier from search results"},
				}),
			},
			Execute: (*DirectSession).referenceRead,
		},
	}
}
