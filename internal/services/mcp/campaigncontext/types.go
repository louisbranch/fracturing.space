package campaigncontext

import "github.com/modelcontextprotocol/go-sdk/mcp"

// ArtifactListInput reads the campaign artifact catalog.
type ArtifactListInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
}

// ArtifactGetInput reads one campaign artifact.
type ArtifactGetInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Path       string `json:"path" jsonschema:"artifact path such as story.md, memory.md, or working/notes.md"`
}

// ArtifactUpsertInput writes one mutable campaign artifact.
type ArtifactUpsertInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Path       string `json:"path" jsonschema:"artifact path such as story.md, memory.md, or working/notes.md"`
	Content    string `json:"content" jsonschema:"full replacement markdown content"`
}

// ArtifactResult mirrors one campaign artifact record.
type ArtifactResult struct {
	CampaignID string `json:"campaign_id"`
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	ReadOnly   bool   `json:"read_only"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// ArtifactListResult returns all persisted artifacts for one campaign.
type ArtifactListResult struct {
	CampaignID string           `json:"campaign_id"`
	Artifacts  []ArtifactResult `json:"artifacts"`
}

// ReferenceSearchInput searches the configured rules corpus.
type ReferenceSearchInput struct {
	System     string `json:"system,omitempty" jsonschema:"system identifier; defaults to daggerheart"`
	Query      string `json:"query" jsonschema:"search query"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"optional max result count"`
}

// ReferenceSearchEntry contains one rules search hit.
type ReferenceSearchEntry struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Snippet    string   `json:"snippet,omitempty"`
}

// ReferenceSearchResult returns one search batch.
type ReferenceSearchResult struct {
	Results []ReferenceSearchEntry `json:"results"`
}

// ReferenceReadInput reads one full rules document.
type ReferenceReadInput struct {
	System     string `json:"system,omitempty" jsonschema:"system identifier; defaults to daggerheart"`
	DocumentID string `json:"document_id" jsonschema:"document identifier from search results"`
}

// ReferenceDocumentResult returns one full rules document.
type ReferenceDocumentResult struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Content    string   `json:"content"`
}

// ArtifactListTool defines the MCP tool schema for listing campaign artifacts.
func ArtifactListTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_list",
		Description: "Lists AI GM campaign artifacts such as skills.md, story.md, memory.md, and working notes",
	}
}

// ArtifactGetTool defines the MCP tool schema for reading one campaign artifact.
func ArtifactGetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_get",
		Description: "Reads one AI GM campaign artifact such as story.md, memory.md, or a working note",
	}
}

// ArtifactUpsertTool defines the MCP tool schema for replacing one mutable campaign artifact.
func ArtifactUpsertTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_upsert",
		Description: "Writes one mutable AI GM campaign artifact such as story.md, memory.md, or a working note",
	}
}

// ReferenceSearchTool defines the MCP tool schema for rules search.
func ReferenceSearchTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "system_reference_search",
		Description: "Searches the configured read-only game-system reference corpus",
	}
}

// ReferenceReadTool defines the MCP tool schema for rules document reads.
func ReferenceReadTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "system_reference_read",
		Description: "Reads one full document from the configured read-only game-system reference corpus",
	}
}

// ArtifactListResourceTemplate defines the campaign artifact list resource template.
func ArtifactListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign_artifact_list",
		Title:       "Campaign Artifacts",
		Description: "Readable listing of AI GM campaign artifacts. URI format: campaign://{campaign_id}/artifacts",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/artifacts",
	}
}

// ArtifactResourceTemplate defines the single artifact resource template.
func ArtifactResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign_artifact",
		Title:       "Campaign Artifact",
		Description: "Readable AI GM campaign artifact. URI format: campaign://{campaign_id}/artifacts/{path}",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/artifacts/{path}",
	}
}
