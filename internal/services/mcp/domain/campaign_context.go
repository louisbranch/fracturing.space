package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CampaignArtifactGetInput reads one campaign artifact.
type CampaignArtifactListInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
}

// CampaignArtifactGetInput reads one campaign artifact.
type CampaignArtifactGetInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Path       string `json:"path" jsonschema:"artifact path such as story.md, memory.md, or working/notes.md"`
}

// CampaignArtifactUpsertInput writes one mutable campaign artifact.
type CampaignArtifactUpsertInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Path       string `json:"path" jsonschema:"artifact path such as story.md, memory.md, or working/notes.md"`
	Content    string `json:"content" jsonschema:"full replacement markdown content"`
}

// CampaignArtifactResult mirrors one campaign artifact record.
type CampaignArtifactResult struct {
	CampaignID string `json:"campaign_id"`
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	ReadOnly   bool   `json:"read_only"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// CampaignArtifactListResult returns all persisted artifacts for one campaign.
type CampaignArtifactListResult struct {
	CampaignID string                   `json:"campaign_id"`
	Artifacts  []CampaignArtifactResult `json:"artifacts"`
}

// SystemReferenceSearchInput searches the configured rules corpus.
type SystemReferenceSearchInput struct {
	System     string `json:"system,omitempty" jsonschema:"system identifier; defaults to daggerheart"`
	Query      string `json:"query" jsonschema:"search query"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"optional max result count"`
}

// SystemReferenceSearchEntry contains one rules search hit.
type SystemReferenceSearchEntry struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Snippet    string   `json:"snippet,omitempty"`
}

// SystemReferenceSearchResult returns one search batch.
type SystemReferenceSearchResult struct {
	Results []SystemReferenceSearchEntry `json:"results"`
}

// SystemReferenceReadInput reads one full rules document.
type SystemReferenceReadInput struct {
	System     string `json:"system,omitempty" jsonschema:"system identifier; defaults to daggerheart"`
	DocumentID string `json:"document_id" jsonschema:"document identifier from search results"`
}

// SystemReferenceDocumentResult returns one full rules document.
type SystemReferenceDocumentResult struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Content    string   `json:"content"`
}

// CampaignArtifactListTool defines the MCP tool schema for listing campaign artifacts.
func CampaignArtifactListTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_list",
		Description: "Lists AI GM campaign artifacts such as skills.md, story.md, memory.md, and working notes",
	}
}

// CampaignArtifactGetTool defines the MCP tool schema for reading one campaign artifact.
func CampaignArtifactGetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_get",
		Description: "Reads one AI GM campaign artifact such as story.md, memory.md, or a working note",
	}
}

// CampaignArtifactUpsertTool defines the MCP tool schema for replacing one mutable campaign artifact.
func CampaignArtifactUpsertTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_artifact_upsert",
		Description: "Writes one mutable AI GM campaign artifact such as story.md, memory.md, or a working note",
	}
}

// SystemReferenceSearchTool defines the MCP tool schema for rules search.
func SystemReferenceSearchTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "system_reference_search",
		Description: "Searches the configured read-only game-system reference corpus",
	}
}

// SystemReferenceReadTool defines the MCP tool schema for rules document reads.
func SystemReferenceReadTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "system_reference_read",
		Description: "Reads one full document from the configured read-only game-system reference corpus",
	}
}

// CampaignArtifactListResourceTemplate defines the campaign artifact list resource template.
func CampaignArtifactListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign_artifact_list",
		Title:       "Campaign Artifacts",
		Description: "Readable listing of AI GM campaign artifacts. URI format: campaign://{campaign_id}/artifacts",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/artifacts",
	}
}

// CampaignArtifactResourceTemplate defines the single artifact resource template.
func CampaignArtifactResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign_artifact",
		Title:       "Campaign Artifact",
		Description: "Readable AI GM campaign artifact. URI format: campaign://{campaign_id}/artifacts/{path}",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/artifacts/{path}",
	}
}

// CampaignArtifactListHandler lists all campaign artifacts for the current campaign.
func CampaignArtifactListHandler(client aiv1.CampaignArtifactServiceClient, getContext func() Context) mcp.ToolHandlerFor[CampaignArtifactListInput, CampaignArtifactListResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignArtifactListInput) (*mcp.CallToolResult, CampaignArtifactListResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignArtifactListResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, CampaignArtifactListResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignArtifactListResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ListCampaignArtifacts(callCtx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
		if err != nil {
			return nil, CampaignArtifactListResult{}, fmt.Errorf("campaign artifact list failed: %w", err)
		}
		result := CampaignArtifactListResult{CampaignID: campaignID}
		for _, artifact := range resp.GetArtifacts() {
			result.Artifacts = append(result.Artifacts, campaignArtifactResultFromProto(artifact, false))
		}
		return CallToolResultWithMetadata(callMeta), result, nil
	}
}

// CampaignArtifactGetHandler reads one campaign artifact.
func CampaignArtifactGetHandler(client aiv1.CampaignArtifactServiceClient, getContext func() Context) mcp.ToolHandlerFor[CampaignArtifactGetInput, CampaignArtifactResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignArtifactGetInput) (*mcp.CallToolResult, CampaignArtifactResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, CampaignArtifactResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
			CampaignId: campaignID,
			Path:       input.Path,
		})
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("campaign artifact get failed: %w", err)
		}
		return CallToolResultWithMetadata(callMeta), campaignArtifactResultFromProto(resp.GetArtifact(), true), nil
	}
}

// CampaignArtifactUpsertHandler writes one mutable campaign artifact.
func CampaignArtifactUpsertHandler(client aiv1.CampaignArtifactServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignArtifactUpsertInput, CampaignArtifactResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignArtifactUpsertInput) (*mcp.CallToolResult, CampaignArtifactResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, CampaignArtifactResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.UpsertCampaignArtifact(callCtx, &aiv1.UpsertCampaignArtifactRequest{
			CampaignId: campaignID,
			Path:       input.Path,
			Content:    input.Content,
		})
		if err != nil {
			return nil, CampaignArtifactResult{}, fmt.Errorf("campaign artifact upsert failed: %w", err)
		}
		result := campaignArtifactResultFromProto(resp.GetArtifact(), true)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/artifacts", campaignID),
			fmt.Sprintf("campaign://%s/artifacts/%s", campaignID, strings.TrimSpace(result.Path)),
		)
		return CallToolResultWithMetadata(callMeta), result, nil
	}
}

// CampaignArtifactListResourceHandler reads the campaign artifact catalog resource.
func CampaignArtifactListResourceHandler(client aiv1.CampaignArtifactServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign artifact client is not configured")
		}
		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/artifacts")
		}
		uri := req.Params.URI
		campaignID, err := parseCampaignIDFromResourceURI(uri, "artifacts")
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}
		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()
		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ListCampaignArtifacts(callCtx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
		if err != nil {
			return nil, fmt.Errorf("campaign artifact list failed: %w", err)
		}
		payload := CampaignArtifactListResult{CampaignID: campaignID}
		for _, artifact := range resp.GetArtifacts() {
			payload.Artifacts = append(payload.Artifacts, campaignArtifactResultFromProto(artifact, false))
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign artifact list: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

// CampaignArtifactResourceHandler reads one campaign artifact resource.
func CampaignArtifactResourceHandler(client aiv1.CampaignArtifactServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign artifact client is not configured")
		}
		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign and artifact path are required; use URI format campaign://{campaign_id}/artifacts/{path}")
		}
		uri := req.Params.URI
		campaignID, artifactPath, err := parseCampaignArtifactResourceURI(uri)
		if err != nil {
			return nil, err
		}
		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()
		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
			CampaignId: campaignID,
			Path:       artifactPath,
		})
		if err != nil {
			return nil, fmt.Errorf("campaign artifact get failed: %w", err)
		}
		data, err := json.MarshalIndent(campaignArtifactResultFromProto(resp.GetArtifact(), true), "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign artifact: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
}

// SystemReferenceSearchHandler searches the configured system reference corpus.
func SystemReferenceSearchHandler(client aiv1.SystemReferenceServiceClient) mcp.ToolHandlerFor[SystemReferenceSearchInput, SystemReferenceSearchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SystemReferenceSearchInput) (*mcp.CallToolResult, SystemReferenceSearchResult, error) {
		callContext, err := newToolInvocationContext(ctx, nil)
		if err != nil {
			return nil, SystemReferenceSearchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, SystemReferenceSearchResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.SearchSystemReference(callCtx, &aiv1.SearchSystemReferenceRequest{
			System:     input.System,
			Query:      input.Query,
			MaxResults: int32(input.MaxResults),
		})
		if err != nil {
			return nil, SystemReferenceSearchResult{}, fmt.Errorf("system reference search failed: %w", err)
		}
		result := SystemReferenceSearchResult{Results: make([]SystemReferenceSearchEntry, 0, len(resp.GetResults()))}
		for _, item := range resp.GetResults() {
			result.Results = append(result.Results, SystemReferenceSearchEntry{
				System:     item.GetSystem(),
				DocumentID: item.GetDocumentId(),
				Title:      item.GetTitle(),
				Kind:       item.GetKind(),
				Path:       item.GetPath(),
				Aliases:    append([]string(nil), item.GetAliases()...),
				Snippet:    item.GetSnippet(),
			})
		}
		return CallToolResultWithMetadata(callMeta), result, nil
	}
}

// SystemReferenceReadHandler reads one full system reference document.
func SystemReferenceReadHandler(client aiv1.SystemReferenceServiceClient) mcp.ToolHandlerFor[SystemReferenceReadInput, SystemReferenceDocumentResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SystemReferenceReadInput) (*mcp.CallToolResult, SystemReferenceDocumentResult, error) {
		callContext, err := newToolInvocationContext(ctx, nil)
		if err != nil {
			return nil, SystemReferenceDocumentResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, SystemReferenceDocumentResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ReadSystemReferenceDocument(callCtx, &aiv1.ReadSystemReferenceDocumentRequest{
			System:     input.System,
			DocumentId: input.DocumentID,
		})
		if err != nil {
			return nil, SystemReferenceDocumentResult{}, fmt.Errorf("system reference read failed: %w", err)
		}
		document := resp.GetDocument()
		return CallToolResultWithMetadata(callMeta), SystemReferenceDocumentResult{
			System:     document.GetSystem(),
			DocumentID: document.GetDocumentId(),
			Title:      document.GetTitle(),
			Kind:       document.GetKind(),
			Path:       document.GetPath(),
			Aliases:    append([]string(nil), document.GetAliases()...),
			Content:    document.GetContent(),
		}, nil
	}
}

func parseCampaignArtifactResourceURI(uri string) (string, string, error) {
	const prefix = "campaign://"
	uri = strings.TrimSpace(uri)
	if !strings.HasPrefix(uri, prefix) {
		return "", "", fmt.Errorf("URI must start with %q", prefix)
	}
	trimmed := strings.TrimPrefix(uri, prefix)
	parts := strings.SplitN(trimmed, "/artifacts/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/artifacts/{path}")
	}
	campaignID := strings.TrimSpace(parts[0])
	artifactPath := strings.TrimSpace(parts[1])
	if campaignID == "" || artifactPath == "" {
		return "", "", fmt.Errorf("campaign and artifact path are required; use URI format campaign://{campaign_id}/artifacts/{path}")
	}
	return campaignID, artifactPath, nil
}

func campaignArtifactResultFromProto(artifact *aiv1.CampaignArtifact, includeContent bool) CampaignArtifactResult {
	if artifact == nil {
		return CampaignArtifactResult{}
	}
	result := CampaignArtifactResult{
		CampaignID: artifact.GetCampaignId(),
		Path:       artifact.GetPath(),
		ReadOnly:   artifact.GetReadOnly(),
		CreatedAt:  formatTimestamp(artifact.GetCreatedAt()),
		UpdatedAt:  formatTimestamp(artifact.GetUpdatedAt()),
	}
	if includeContent {
		result.Content = artifact.GetContent()
	}
	return result
}
