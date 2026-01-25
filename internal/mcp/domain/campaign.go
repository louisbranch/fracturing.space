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
	ThemePrompt string `json:"theme_prompt,omitempty" jsonschema:"optional theme prompt"`
}

// CampaignCreateResult represents the MCP tool output for campaign creation.
type CampaignCreateResult struct {
	ID          string `json:"id" jsonschema:"campaign identifier"`
	Name        string `json:"name" jsonschema:"campaign name"`
	GmMode      string `json:"gm_mode" jsonschema:"gm mode"`
	PlayerCount int    `json:"player_count" jsonschema:"number of registered players"`
	ThemePrompt string `json:"theme_prompt" jsonschema:"theme prompt"`
}

// CampaignListEntry represents a readable campaign metadata entry.
type CampaignListEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	GmMode      string `json:"gm_mode"`
	PlayerCount int    `json:"player_count"`
	ThemePrompt string `json:"theme_prompt"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CampaignListPayload represents the MCP resource payload for campaign listings.
type CampaignListPayload struct {
	Campaigns []CampaignListEntry `json:"campaigns"`
}

// ParticipantCreateInput represents the MCP tool input for participant creation.
type ParticipantCreateInput struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	DisplayName string `json:"display_name" jsonschema:"display name for the participant"`
	Role        string `json:"role" jsonschema:"participant role (GM, PLAYER)"`
	Controller  string `json:"controller,omitempty" jsonschema:"controller type (HUMAN, AI); optional, defaults to HUMAN if unspecified"`
}

// ParticipantCreateResult represents the MCP tool output for participant creation.
type ParticipantCreateResult struct {
	ID          string `json:"id" jsonschema:"participant identifier"`
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	DisplayName string `json:"display_name" jsonschema:"display name for the participant"`
	Role        string `json:"role" jsonschema:"participant role"`
	Controller  string `json:"controller" jsonschema:"controller type"`
	CreatedAt   string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt   string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// ParticipantListEntry represents a readable participant entry.
type ParticipantListEntry struct {
	ID          string `json:"id"`
	CampaignID  string `json:"campaign_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Controller  string `json:"controller"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ParticipantListPayload represents the MCP resource payload for participant listings.
type ParticipantListPayload struct {
	Participants []ParticipantListEntry `json:"participants"`
}

// ActorListEntry represents a readable actor entry.
type ActorListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Notes      string `json:"notes"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ActorListPayload represents the MCP resource payload for actor listings.
type ActorListPayload struct {
	Actors []ActorListEntry `json:"actors"`
}

// ActorCreateInput represents the MCP tool input for actor creation.
type ActorCreateInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the actor"`
	Kind       string `json:"kind" jsonschema:"actor kind (PC, NPC)"`
	Notes      string `json:"notes,omitempty" jsonschema:"optional free-form notes about the actor"`
}

// ActorCreateResult represents the MCP tool output for actor creation.
type ActorCreateResult struct {
	ID         string `json:"id" jsonschema:"actor identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the actor"`
	Kind       string `json:"kind" jsonschema:"actor kind"`
	Notes      string `json:"notes" jsonschema:"free-form notes about the actor"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when actor was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when actor was last updated"`
}

// ActorControlSetInput represents the MCP tool input for setting actor control.
type ActorControlSetInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	ActorID    string `json:"actor_id" jsonschema:"actor identifier"`
	Controller string `json:"controller" jsonschema:"controller: 'GM' (case-insensitive) for GM control, or a participant ID for participant control"`
}

// ActorControlSetResult represents the MCP tool output for setting actor control.
type ActorControlSetResult struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	ActorID    string `json:"actor_id" jsonschema:"actor identifier"`
	Controller string `json:"controller" jsonschema:"controller: 'GM' or the participant ID"`
}

// CampaignCreateTool defines the MCP tool schema for creating campaigns.
func CampaignCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_create",
		Description: "Creates a new campaign metadata record",
	}
}

// ParticipantCreateTool defines the MCP tool schema for creating participants.
func ParticipantCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "participant_create",
		Description: "Creates a participant (GM or player) for a campaign",
	}
}

// ActorCreateTool defines the MCP tool schema for creating actors.
func ActorCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "actor_create",
		Description: "Creates an actor (PC or NPC) for a campaign",
	}
}

// ActorControlSetTool defines the MCP tool schema for setting actor control.
func ActorControlSetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "actor_control_set",
		Description: "Sets the default controller (GM or participant) for an actor in a campaign",
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

// ParticipantListResource defines the MCP resource for participant listings.
// The effective URI template is campaign://{campaign_id}/participants, but the
// SDK requires a valid URI for registration, so we use a placeholder here.
// Clients must provide the full URI with actual campaign_id when reading.
func ParticipantListResource() *mcp.Resource {
	return &mcp.Resource{
		Name:        "participant_list",
		Title:       "Participants",
		Description: "Readable listing of participants for a campaign. URI format: campaign://{campaign_id}/participants",
		MIMEType:    "application/json",
		URI:         "campaign://_/participants", // Placeholder; actual format: campaign://{campaign_id}/participants
	}
}

// ActorListResource defines the MCP resource for actor listings.
// The effective URI template is campaign://{campaign_id}/actors, but the
// SDK requires a valid URI for registration, so we use a placeholder here.
// Clients must provide the full URI with actual campaign_id when reading.
func ActorListResource() *mcp.Resource {
	return &mcp.Resource{
		Name:        "actor_list",
		Title:       "Actors",
		Description: "Readable listing of actors for a campaign. URI format: campaign://{campaign_id}/actors",
		MIMEType:    "application/json",
		URI:         "campaign://_/actors", // Placeholder; actual format: campaign://{campaign_id}/actors
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
			PlayerCount: int(response.Campaign.GetPlayerCount()),
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
				PlayerCount: int(campaign.GetPlayerCount()),
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

// ParticipantCreateHandler executes a participant creation request.
func ParticipantCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[ParticipantCreateInput, ParticipantCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantCreateInput) (*mcp.CallToolResult, ParticipantCreateResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req := &campaignv1.CreateParticipantRequest{
			CampaignId:  input.CampaignID,
			DisplayName: input.DisplayName,
			Role:        participantRoleFromString(input.Role),
		}

		// Controller is optional; only set if provided
		if input.Controller != "" {
			req.Controller = controllerFromString(input.Controller)
		}

		response, err := client.CreateParticipant(runCtx, req)
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

// ActorCreateHandler executes an actor creation request.
func ActorCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[ActorCreateInput, ActorCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ActorCreateInput) (*mcp.CallToolResult, ActorCreateResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req := &campaignv1.CreateActorRequest{
			CampaignId: input.CampaignID,
			Name:       input.Name,
			Kind:       actorKindFromString(input.Kind),
			Notes:      input.Notes,
		}

		response, err := client.CreateActor(runCtx, req)
		if err != nil {
			return nil, ActorCreateResult{}, fmt.Errorf("actor create failed: %w", err)
		}
		if response == nil || response.Actor == nil {
			return nil, ActorCreateResult{}, fmt.Errorf("actor create response is missing")
		}

		result := ActorCreateResult{
			ID:         response.Actor.GetId(),
			CampaignID: response.Actor.GetCampaignId(),
			Name:       response.Actor.GetName(),
			Kind:       actorKindToString(response.Actor.GetKind()),
			Notes:      response.Actor.GetNotes(),
			CreatedAt:  formatTimestamp(response.Actor.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Actor.GetUpdatedAt()),
		}

		return nil, result, nil
	}
}

func actorKindFromString(value string) campaignv1.ActorKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return campaignv1.ActorKind_PC
	case "NPC":
		return campaignv1.ActorKind_NPC
	default:
		return campaignv1.ActorKind_ACTOR_KIND_UNSPECIFIED
	}
}

func actorKindToString(kind campaignv1.ActorKind) string {
	switch kind {
	case campaignv1.ActorKind_PC:
		return "PC"
	case campaignv1.ActorKind_NPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

// ActorControlSetHandler executes an actor control set request.
func ActorControlSetHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[ActorControlSetInput, ActorControlSetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ActorControlSetInput) (*mcp.CallToolResult, ActorControlSetResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		controller, err := actorControllerFromString(input.Controller)
		if err != nil {
			return nil, ActorControlSetResult{}, fmt.Errorf("invalid controller: %w", err)
		}

		req := &campaignv1.SetDefaultControlRequest{
			CampaignId: input.CampaignID,
			ActorId:    input.ActorID,
			Controller: controller,
		}

		response, err := client.SetDefaultControl(runCtx, req)
		if err != nil {
			return nil, ActorControlSetResult{}, fmt.Errorf("actor control set failed: %w", err)
		}
		if response == nil {
			return nil, ActorControlSetResult{}, fmt.Errorf("actor control set response is missing")
		}

		result := ActorControlSetResult{
			CampaignID: response.GetCampaignId(),
			ActorID:    response.GetActorId(),
			Controller: actorControllerToString(response.GetController()),
		}

		return nil, result, nil
	}
}

// actorControllerFromString converts a string to a protobuf ActorController.
// Accepts "GM" (case-insensitive) for GM control, or a participant ID for participant control.
func actorControllerFromString(controller string) (*campaignv1.ActorController, error) {
	controller = strings.TrimSpace(controller)
	if controller == "" {
		return nil, fmt.Errorf("controller is required")
	}

	upper := strings.ToUpper(controller)
	if upper == "GM" {
		return &campaignv1.ActorController{
			Controller: &campaignv1.ActorController_Gm{
				Gm: &campaignv1.GmController{},
			},
		}, nil
	}

	// Otherwise, treat as participant ID
	return &campaignv1.ActorController{
		Controller: &campaignv1.ActorController_Participant{
			Participant: &campaignv1.ParticipantController{
				ParticipantId: controller,
			},
		},
	}, nil
}

// actorControllerToString converts a protobuf ActorController to a string representation.
// Returns "GM" for GM control, or the participant ID for participant control.
func actorControllerToString(controller *campaignv1.ActorController) string {
	if controller == nil {
		return ""
	}

	switch c := controller.GetController().(type) {
	case *campaignv1.ActorController_Gm:
		return "GM"
	case *campaignv1.ActorController_Participant:
		if c.Participant != nil {
			return c.Participant.GetParticipantId()
		}
		return ""
	default:
		return ""
	}
}

// ParticipantListResourceHandler returns a readable participant listing resource.
func ParticipantListResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("participant list client is not configured")
		}

		uri := ParticipantListResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/participants.
		// If the URI is the registered placeholder, return an error requiring a concrete campaign ID.
		// Otherwise, parse the campaign ID from the URI path.
		var campaignID string
		var err error
		if uri == ParticipantListResource().URI {
			// Using registered placeholder URI - this shouldn't happen in practice
			// but handle it gracefully by requiring campaign_id in a different way
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/participants")
		}
		campaignID, err = parseCampaignIDFromURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		payload := ParticipantListPayload{}
		response, err := client.ListParticipants(runCtx, &campaignv1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("participant list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("participant list response is missing")
		}

		for _, participant := range response.GetParticipants() {
			payload.Participants = append(payload.Participants, ParticipantListEntry{
				ID:          participant.GetId(),
				CampaignID:  participant.GetCampaignId(),
				DisplayName: participant.GetDisplayName(),
				Role:        participantRoleToString(participant.GetRole()),
				Controller:  controllerToString(participant.GetController()),
				CreatedAt:   formatTimestamp(participant.GetCreatedAt()),
				UpdatedAt:   formatTimestamp(participant.GetUpdatedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal participant list: %w", err)
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

// parseCampaignIDFromURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/participants.
// It parses URIs of the expected format but requires an actual campaign ID and rejects the placeholder (campaign://_/participants).
func parseCampaignIDFromURI(uri string) (string, error) {
	prefix := "campaign://"
	suffix := "/participants"

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, suffix) {
		return "", fmt.Errorf("URI must end with %q", suffix)
	}

	campaignID := strings.TrimPrefix(uri, prefix)
	campaignID = strings.TrimSuffix(campaignID, suffix)
	campaignID = strings.TrimSpace(campaignID)

	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}

	// Reject the placeholder value - actual campaign IDs must be provided
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}

	return campaignID, nil
}

// ActorListResourceHandler returns a readable actor listing resource.
func ActorListResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("actor list client is not configured")
		}

		uri := ActorListResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/actors.
		// If the URI is the registered placeholder, return an error requiring a concrete campaign ID.
		// Otherwise, parse the campaign ID from the URI path.
		var campaignID string
		var err error
		if uri == ActorListResource().URI {
			// Using registered placeholder URI - this shouldn't happen in practice
			// but handle it gracefully by requiring campaign_id in a different way
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/actors")
		}
		campaignID, err = parseCampaignIDFromActorURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		payload := ActorListPayload{}
		response, err := client.ListActors(runCtx, &campaignv1.ListActorsRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("actor list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("actor list response is missing")
		}

		for _, actor := range response.GetActors() {
			payload.Actors = append(payload.Actors, ActorListEntry{
				ID:         actor.GetId(),
				CampaignID: actor.GetCampaignId(),
				Name:       actor.GetName(),
				Kind:       actorKindToString(actor.GetKind()),
				Notes:      actor.GetNotes(),
				CreatedAt:  formatTimestamp(actor.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(actor.GetUpdatedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal actor list: %w", err)
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

// parseCampaignIDFromActorURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/actors.
// It parses URIs of the expected format but requires an actual campaign ID and rejects the placeholder (campaign://_/actors).
func parseCampaignIDFromActorURI(uri string) (string, error) {
	prefix := "campaign://"
	suffix := "/actors"

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}
	if !strings.HasSuffix(uri, suffix) {
		return "", fmt.Errorf("URI must end with %q", suffix)
	}

	campaignID := strings.TrimPrefix(uri, prefix)
	campaignID = strings.TrimSuffix(campaignID, suffix)
	campaignID = strings.TrimSpace(campaignID)

	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}

	// Reject the placeholder value - actual campaign IDs must be provided
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}

	return campaignID, nil
}
