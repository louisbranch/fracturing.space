package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// SceneCreateInput represents the MCP tool input for creating a scene.
type SceneCreateInput struct {
	CampaignID   string   `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SessionID    string   `json:"session_id,omitempty" jsonschema:"session identifier (defaults to context)"`
	Name         string   `json:"name" jsonschema:"scene title"`
	Description  string   `json:"description,omitempty" jsonschema:"scene framing description"`
	CharacterIDs []string `json:"character_ids,omitempty" jsonschema:"optional starting character identifiers"`
}

// SceneCreateResult represents the MCP tool output for creating a scene.
type SceneCreateResult struct {
	SceneID      string   `json:"scene_id"`
	CampaignID   string   `json:"campaign_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

// SceneListEntry represents one readable scene entry.
type SceneListEntry struct {
	SceneID      string   `json:"scene_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Active       bool     `json:"active"`
	CharacterIDs []string `json:"character_ids,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	EndedAt      string   `json:"ended_at,omitempty"`
}

// SceneListPayload represents the MCP resource payload for session scenes.
type SceneListPayload struct {
	Scenes []SceneListEntry `json:"scenes"`
}

// SceneCreateTool defines the MCP tool schema for creating scenes.
func SceneCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "scene_create",
		Description: "Creates a new scene within the current session; creation alone does not make it the authoritative active scene",
	}
}

// SceneListResourceTemplate defines the readable session scene-list resource.
func SceneListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "scene_list",
		Title:       "Scenes",
		Description: "Readable listing of scenes for a session. URI format: campaign://{campaign_id}/sessions/{session_id}/scenes",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/sessions/{session_id}/scenes",
	}
}

// SceneCreateHandler creates scenes through MCP.
func SceneCreateHandler(client statev1.SceneServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[SceneCreateInput, SceneCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SceneCreateInput) (*mcp.CallToolResult, SceneCreateResult, error) {
		callCtx, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, SceneCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callCtx.Cancel()

		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			campaignID = strings.TrimSpace(callCtx.MCPContext.CampaignID)
		}
		sessionID := strings.TrimSpace(input.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(callCtx.MCPContext.SessionID)
		}
		if campaignID == "" {
			return nil, SceneCreateResult{}, fmt.Errorf("campaign_id is required")
		}
		if sessionID == "" {
			return nil, SceneCreateResult{}, fmt.Errorf("session_id is required")
		}

		outCtx, callMeta, err := NewOutgoingContextWithContext(callCtx.RunCtx, callCtx.InvocationID, callCtx.MCPContext)
		if err != nil {
			return nil, SceneCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		res, err := client.CreateScene(outCtx, &statev1.CreateSceneRequest{
			CampaignId:   campaignID,
			SessionId:    sessionID,
			Name:         strings.TrimSpace(input.Name),
			Description:  strings.TrimSpace(input.Description),
			CharacterIds: append([]string(nil), input.CharacterIDs...),
		}, grpc.Header(&header))
		if err != nil {
			return nil, SceneCreateResult{}, fmt.Errorf("create scene failed: %w", err)
		}
		if res == nil || strings.TrimSpace(res.GetSceneId()) == "" {
			return nil, SceneCreateResult{}, fmt.Errorf("create scene response is missing")
		}

		result := SceneCreateResult{
			SceneID:      strings.TrimSpace(res.GetSceneId()),
			CampaignID:   campaignID,
			SessionID:    sessionID,
			Name:         strings.TrimSpace(input.Name),
			Description:  strings.TrimSpace(input.Description),
			CharacterIDs: append([]string(nil), input.CharacterIDs...),
		}
		meta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(ctx, notify, fmt.Sprintf("campaign://%s/sessions/%s/scenes", campaignID, sessionID))
		return CallToolResultWithMetadata(meta), result, nil
	}
}

// SceneListResourceHandler returns a readable session scene listing resource.
func SceneListResourceHandler(client statev1.SceneServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("scene list client is not configured")
		}
		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign and session IDs are required; use URI format campaign://{campaign_id}/sessions/{session_id}/scenes")
		}
		uri := strings.TrimSpace(req.Params.URI)
		campaignID, sessionID, err := parseSceneListURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse scene list uri: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		res, err := client.ListScenes(callCtx, &statev1.ListScenesRequest{
			CampaignId: campaignID,
			SessionId:  sessionID,
			PageSize:   20,
		})
		if err != nil {
			return nil, fmt.Errorf("list scenes failed: %w", err)
		}
		if res == nil {
			return nil, fmt.Errorf("scene list response is missing")
		}

		payload := SceneListPayload{Scenes: make([]SceneListEntry, 0, len(res.GetScenes()))}
		for _, sc := range res.GetScenes() {
			entry := SceneListEntry{
				SceneID:      sc.GetSceneId(),
				SessionID:    sc.GetSessionId(),
				Name:         sc.GetName(),
				Description:  sc.GetDescription(),
				Active:       sc.GetActive(),
				CharacterIDs: append([]string(nil), sc.GetCharacterIds()...),
				CreatedAt:    formatTimestamp(sc.GetCreatedAt()),
				UpdatedAt:    formatTimestamp(sc.GetUpdatedAt()),
			}
			if sc.GetEndedAt() != nil {
				entry.EndedAt = formatTimestamp(sc.GetEndedAt())
			}
			payload.Scenes = append(payload.Scenes, entry)
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal scene list: %w", err)
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

func parseSceneListURI(uri string) (string, string, error) {
	trimmed := strings.TrimSpace(uri)
	if !strings.HasPrefix(trimmed, "campaign://") {
		return "", "", fmt.Errorf("URI must start with %q", "campaign://")
	}
	rest := strings.TrimPrefix(trimmed, "campaign://")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || strings.TrimSpace(parts[1]) != "sessions" || strings.TrimSpace(parts[3]) != "scenes" {
		return "", "", fmt.Errorf("URI must match campaign://{campaign_id}/sessions/{session_id}/scenes")
	}
	campaignID := strings.TrimSpace(parts[0])
	sessionID := strings.TrimSpace(parts[2])
	if campaignID == "" {
		return "", "", fmt.Errorf("campaign ID is required in URI")
	}
	if campaignID == "_" {
		return "", "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}
	if sessionID == "" {
		return "", "", fmt.Errorf("session ID is required in URI")
	}
	if sessionID == "_" {
		return "", "", fmt.Errorf("session ID placeholder '_' is not a valid session ID")
	}
	if strings.ContainsAny(campaignID, "?#") || strings.ContainsAny(sessionID, "?#") {
		return "", "", fmt.Errorf("URI must not contain query parameters or fragments")
	}
	return campaignID, sessionID, nil
}
