package campaigncontext

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ArtifactListHandler lists all campaign artifacts for the current campaign.
func ArtifactListHandler(client aiv1.CampaignArtifactServiceClient, getContext func() sessionctx.Context) mcp.ToolHandlerFor[ArtifactListInput, ArtifactListResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ArtifactListInput) (*mcp.CallToolResult, ArtifactListResult, error) {
		callContext, err := sessionctx.NewToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ArtifactListResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, ArtifactListResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := sessionctx.NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ArtifactListResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ListCampaignArtifacts(callCtx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
		if err != nil {
			return nil, ArtifactListResult{}, fmt.Errorf("campaign artifact list failed: %w", err)
		}
		result := ArtifactListResult{CampaignID: campaignID}
		for _, artifact := range resp.GetArtifacts() {
			result.Artifacts = append(result.Artifacts, artifactResultFromProto(artifact, false))
		}
		return sessionctx.CallToolResultWithMetadata(callMeta), result, nil
	}
}

// ArtifactGetHandler reads one campaign artifact.
func ArtifactGetHandler(client aiv1.CampaignArtifactServiceClient, getContext func() sessionctx.Context) mcp.ToolHandlerFor[ArtifactGetInput, ArtifactResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ArtifactGetInput) (*mcp.CallToolResult, ArtifactResult, error) {
		callContext, err := sessionctx.NewToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, ArtifactResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := sessionctx.NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
			CampaignId: campaignID,
			Path:       input.Path,
		})
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("campaign artifact get failed: %w", err)
		}
		return sessionctx.CallToolResultWithMetadata(callMeta), artifactResultFromProto(resp.GetArtifact(), true), nil
	}
}

// ArtifactUpsertHandler writes one mutable campaign artifact.
func ArtifactUpsertHandler(client aiv1.CampaignArtifactServiceClient, getContext func() sessionctx.Context, notify sessionctx.ResourceUpdateNotifier) mcp.ToolHandlerFor[ArtifactUpsertInput, ArtifactResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ArtifactUpsertInput) (*mcp.CallToolResult, ArtifactResult, error) {
		callContext, err := sessionctx.NewToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
		}
		if campaignID == "" {
			return nil, ArtifactResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := sessionctx.NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.UpsertCampaignArtifact(callCtx, &aiv1.UpsertCampaignArtifactRequest{
			CampaignId: campaignID,
			Path:       input.Path,
			Content:    input.Content,
		})
		if err != nil {
			return nil, ArtifactResult{}, fmt.Errorf("campaign artifact upsert failed: %w", err)
		}
		result := artifactResultFromProto(resp.GetArtifact(), true)
		sessionctx.NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/artifacts", campaignID),
			fmt.Sprintf("campaign://%s/artifacts/%s", campaignID, strings.TrimSpace(result.Path)),
		)
		return sessionctx.CallToolResultWithMetadata(callMeta), result, nil
	}
}

// ArtifactListResourceHandler reads the campaign artifact catalog resource.
func ArtifactListResourceHandler(client aiv1.CampaignArtifactServiceClient) mcp.ResourceHandler {
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
		runCtx, cancel := context.WithTimeout(ctx, sessionctx.CallTimeout)
		defer cancel()
		callCtx, _, err := sessionctx.NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ListCampaignArtifacts(callCtx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
		if err != nil {
			return nil, fmt.Errorf("campaign artifact list failed: %w", err)
		}
		payload := ArtifactListResult{CampaignID: campaignID}
		for _, artifact := range resp.GetArtifacts() {
			payload.Artifacts = append(payload.Artifacts, artifactResultFromProto(artifact, false))
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

// ArtifactResourceHandler reads one campaign artifact resource.
func ArtifactResourceHandler(client aiv1.CampaignArtifactServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign artifact client is not configured")
		}
		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign and artifact path are required; use URI format campaign://{campaign_id}/artifacts/{path}")
		}
		uri := req.Params.URI
		campaignID, artifactPath, err := parseArtifactResourceURI(uri)
		if err != nil {
			return nil, err
		}
		runCtx, cancel := context.WithTimeout(ctx, sessionctx.CallTimeout)
		defer cancel()
		callCtx, _, err := sessionctx.NewOutgoingContext(runCtx, "")
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
		data, err := json.MarshalIndent(artifactResultFromProto(resp.GetArtifact(), true), "", "  ")
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

// ReferenceSearchHandler searches the configured system reference corpus.
func ReferenceSearchHandler(client aiv1.SystemReferenceServiceClient) mcp.ToolHandlerFor[ReferenceSearchInput, ReferenceSearchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ReferenceSearchInput) (*mcp.CallToolResult, ReferenceSearchResult, error) {
		callContext, err := sessionctx.NewToolInvocationContext(ctx, nil)
		if err != nil {
			return nil, ReferenceSearchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := sessionctx.NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, ReferenceSearchResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.SearchSystemReference(callCtx, &aiv1.SearchSystemReferenceRequest{
			System:     input.System,
			Query:      input.Query,
			MaxResults: int32(input.MaxResults),
		})
		if err != nil {
			return nil, ReferenceSearchResult{}, fmt.Errorf("system reference search failed: %w", err)
		}
		result := ReferenceSearchResult{Results: make([]ReferenceSearchEntry, 0, len(resp.GetResults()))}
		for _, item := range resp.GetResults() {
			result.Results = append(result.Results, ReferenceSearchEntry{
				System:     item.GetSystem(),
				DocumentID: item.GetDocumentId(),
				Title:      item.GetTitle(),
				Kind:       item.GetKind(),
				Path:       item.GetPath(),
				Aliases:    append([]string(nil), item.GetAliases()...),
				Snippet:    item.GetSnippet(),
			})
		}
		return sessionctx.CallToolResultWithMetadata(callMeta), result, nil
	}
}

// ReferenceReadHandler reads one full system reference document.
func ReferenceReadHandler(client aiv1.SystemReferenceServiceClient) mcp.ToolHandlerFor[ReferenceReadInput, ReferenceDocumentResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ReferenceReadInput) (*mcp.CallToolResult, ReferenceDocumentResult, error) {
		callContext, err := sessionctx.NewToolInvocationContext(ctx, nil)
		if err != nil {
			return nil, ReferenceDocumentResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := sessionctx.NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, ReferenceDocumentResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		resp, err := client.ReadSystemReferenceDocument(callCtx, &aiv1.ReadSystemReferenceDocumentRequest{
			System:     input.System,
			DocumentId: input.DocumentID,
		})
		if err != nil {
			return nil, ReferenceDocumentResult{}, fmt.Errorf("system reference read failed: %w", err)
		}
		document := resp.GetDocument()
		return sessionctx.CallToolResultWithMetadata(callMeta), ReferenceDocumentResult{
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

func parseCampaignIDFromResourceURI(uri, resourceType string) (string, error) {
	const prefix = "campaign://"
	suffix := "/" + resourceType

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, suffix) {
		return "", fmt.Errorf("URI must end with %q", suffix)
	}

	campaignID := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(uri, prefix), suffix))
	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}
	return campaignID, nil
}

func parseArtifactResourceURI(uri string) (string, string, error) {
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

func artifactResultFromProto(artifact *aiv1.CampaignArtifact, includeContent bool) ArtifactResult {
	if artifact == nil {
		return ArtifactResult{}
	}
	result := ArtifactResult{
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

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}
