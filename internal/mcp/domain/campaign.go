package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CampaignCreateInput represents the MCP tool input for campaign creation.
type CampaignCreateInput struct {
	Name        string `json:"name" jsonschema:"campaign name"`
	GmMode      string `json:"gm_mode" jsonschema:"gm mode (HUMAN, AI, HYBRID)"`
	PlayerSlots int    `json:"player_slots" jsonschema:"number of player slots"`
	ThemePrompt string `json:"theme_prompt" jsonschema:"optional theme prompt"`
}

// CampaignCreateResult represents the MCP tool output for campaign creation.
type CampaignCreateResult struct {
	ID          string `json:"id" jsonschema:"campaign identifier"`
	Name        string `json:"name" jsonschema:"campaign name"`
	GmMode      string `json:"gm_mode" jsonschema:"gm mode"`
	PlayerSlots int    `json:"player_slots" jsonschema:"number of player slots"`
	ThemePrompt string `json:"theme_prompt" jsonschema:"theme prompt"`
}

// CampaignListEntry represents a readable campaign metadata entry.
type CampaignListEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	GmMode      string `json:"gm_mode"`
	PlayerSlots int    `json:"player_slots"`
	ThemePrompt string `json:"theme_prompt"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CampaignListPayload represents the MCP resource payload for campaign listings.
type CampaignListPayload struct {
	Campaigns []CampaignListEntry `json:"campaigns"`
}

// CampaignCreateTool defines the MCP tool schema for creating campaigns.
func CampaignCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_create",
		Description: "Creates a new campaign metadata record",
	}
}

// CampaignListResource defines the MCP resource for campaign listings.
func CampaignListResource() *mcp.Resource {
	return &mcp.Resource{
		Name:        "campaign_list",
		Title:       "Campaigns",
		Description: "Readable listing of campaign metadata records",
		MIMEType:    "application/json",
		URI:         "campaigns://list",
	}
}

// CampaignCreateHandler executes a campaign creation request.
func CampaignCreateHandler(client campaignpb.CampaignServiceClient) mcp.ToolHandlerFor[CampaignCreateInput, CampaignCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignCreateInput) (*mcp.CallToolResult, CampaignCreateResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.CreateCampaign(runCtx, &campaignpb.CreateCampaignRequest{
			Name:        input.Name,
			GmMode:      gmModeFromString(input.GmMode),
			PlayerSlots: int32(input.PlayerSlots),
			ThemePrompt: input.ThemePrompt,
		})
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response is missing")
		}

		result := CampaignCreateResult{
			ID:          response.Campaign.GetId(),
			Name:        response.Campaign.GetName(),
			GmMode:      gmModeToString(response.Campaign.GetGmMode()),
			PlayerSlots: int(response.Campaign.GetPlayerSlots()),
			ThemePrompt: response.Campaign.GetThemePrompt(),
		}

		return nil, result, nil
	}
}

// CampaignListResourceHandler returns a readable campaign listing resource.
func CampaignListResourceHandler(client campaignpb.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign list client is not configured")
		}

		uri := CampaignListResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		payload := CampaignListPayload{}
		pageToken := ""
		for {
			response, err := client.ListCampaigns(runCtx, &campaignpb.ListCampaignsRequest{
				PageSize:  10,
				PageToken: pageToken,
			})
			if err != nil {
				return nil, fmt.Errorf("campaign list failed: %w", err)
			}
			if response == nil {
				return nil, fmt.Errorf("campaign list response is missing")
			}

			for _, campaign := range response.GetCampaigns() {
				payload.Campaigns = append(payload.Campaigns, CampaignListEntry{
					ID:          campaign.GetId(),
					Name:        campaign.GetName(),
					GmMode:      gmModeToString(campaign.GetGmMode()),
					PlayerSlots: int(campaign.GetPlayerSlots()),
					ThemePrompt: campaign.GetThemePrompt(),
					CreatedAt:   formatTimestamp(campaign.GetCreatedAt()),
					UpdatedAt:   formatTimestamp(campaign.GetUpdatedAt()),
				})
			}

			if response.GetNextPageToken() == "" {
				break
			}
			pageToken = response.GetNextPageToken()
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign list: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}

// formatTimestamp returns an RFC3339 timestamp or empty string.
func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

func gmModeFromString(value string) campaignpb.GmMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return campaignpb.GmMode_HUMAN
	case "AI":
		return campaignpb.GmMode_AI
	case "HYBRID":
		return campaignpb.GmMode_HYBRID
	default:
		return campaignpb.GmMode_GM_MODE_UNSPECIFIED
	}
}

func gmModeToString(mode campaignpb.GmMode) string {
	switch mode {
	case campaignpb.GmMode_HUMAN:
		return "HUMAN"
	case campaignpb.GmMode_AI:
		return "AI"
	case campaignpb.GmMode_HYBRID:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}
