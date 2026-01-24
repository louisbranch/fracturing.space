package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CampaignCreateInput represents the MCP tool input for campaign creation.
type CampaignCreateInput struct {
	Name        string `json:"name" jsonschema:"campaign name"`
	GmMode      string `json:"gm_mode" jsonschema:"gm mode (HUMAN, AI, HYBRID)"`
	PlayerSlots int    `json:"player_slots" jsonschema:"number of player slots"`
	ThemePrompt string `json:"theme_prompt,omitempty" jsonschema:"optional theme prompt"`
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

// ParticipantCreateInput represents the MCP tool input for participant registration.
type ParticipantCreateInput struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	DisplayName string `json:"display_name" jsonschema:"display name for the participant"`
	Role        string `json:"role" jsonschema:"participant role (GM, PLAYER)"`
	Controller  string `json:"controller,omitempty" jsonschema:"controller type (HUMAN, AI); optional, defaults to HUMAN if unspecified"`
}

// ParticipantCreateResult represents the MCP tool output for participant registration.
type ParticipantCreateResult struct {
	ID          string `json:"id" jsonschema:"participant identifier"`
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	DisplayName string `json:"display_name" jsonschema:"display name for the participant"`
	Role        string `json:"role" jsonschema:"participant role"`
	Controller  string `json:"controller" jsonschema:"controller type"`
	CreatedAt   string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt   string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// CampaignCreateTool defines the MCP tool schema for creating campaigns.
func CampaignCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_create",
		Description: "Creates a new campaign metadata record",
	}
}

// ParticipantCreateTool defines the MCP tool schema for registering participants.
func ParticipantCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "participant_create",
		Description: "Registers a participant (GM or player) for a campaign",
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
func CampaignCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[CampaignCreateInput, CampaignCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignCreateInput) (*mcp.CallToolResult, CampaignCreateResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.CreateCampaign(runCtx, &campaignv1.CreateCampaignRequest{
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
func CampaignListResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
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
		// TODO: Support page_size/page_token inputs and return next_page_token.
		response, err := client.ListCampaigns(runCtx, &campaignv1.ListCampaignsRequest{
			PageSize: 10,
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

func gmModeFromString(value string) campaignv1.GmMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return campaignv1.GmMode_HUMAN
	case "AI":
		return campaignv1.GmMode_AI
	case "HYBRID":
		return campaignv1.GmMode_HYBRID
	default:
		return campaignv1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func gmModeToString(mode campaignv1.GmMode) string {
	switch mode {
	case campaignv1.GmMode_HUMAN:
		return "HUMAN"
	case campaignv1.GmMode_AI:
		return "AI"
	case campaignv1.GmMode_HYBRID:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

// ParticipantCreateHandler executes a participant registration request.
func ParticipantCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[ParticipantCreateInput, ParticipantCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantCreateInput) (*mcp.CallToolResult, ParticipantCreateResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req := &campaignv1.RegisterParticipantRequest{
			CampaignId:  input.CampaignID,
			DisplayName: input.DisplayName,
			Role:        participantRoleFromString(input.Role),
		}

		// Controller is optional; only set if provided
		if input.Controller != "" {
			req.Controller = controllerFromString(input.Controller)
		}

		response, err := client.RegisterParticipant(runCtx, req)
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("participant create failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("participant create response is missing")
		}

		result := ParticipantCreateResult{
			ID:          response.Participant.GetId(),
			CampaignID:  response.Participant.GetCampaignId(),
			DisplayName: response.Participant.GetDisplayName(),
			Role:        participantRoleToString(response.Participant.GetRole()),
			Controller:  controllerToString(response.Participant.GetController()),
			CreatedAt:   formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:   formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		return nil, result, nil
	}
}

func participantRoleFromString(value string) campaignv1.ParticipantRole {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GM":
		return campaignv1.ParticipantRole_GM
	case "PLAYER":
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

func participantRoleToString(role campaignv1.ParticipantRole) string {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return "GM"
	case campaignv1.ParticipantRole_PLAYER:
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

func controllerFromString(value string) campaignv1.Controller {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return campaignv1.Controller_CONTROLLER_HUMAN
	case "AI":
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}

func controllerToString(controller campaignv1.Controller) string {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return "HUMAN"
	case campaignv1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}
