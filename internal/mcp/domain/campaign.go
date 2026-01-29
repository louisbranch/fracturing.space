package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	ID               string `json:"id" jsonschema:"campaign identifier"`
	Name             string `json:"name" jsonschema:"campaign name"`
	GmMode           string `json:"gm_mode" jsonschema:"gm mode"`
	ParticipantCount int    `json:"participant_count" jsonschema:"number of all participants (GM + PLAYER + future roles)"`
	CharacterCount   int    `json:"character_count" jsonschema:"number of all characters (PC + NPC + future kinds)"`
	GmFear           int    `json:"gm_fear" jsonschema:"campaign-scoped GM fear"`
	ThemePrompt      string `json:"theme_prompt" jsonschema:"theme prompt"`
}

// CampaignListEntry represents a readable campaign metadata entry.
type CampaignListEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	GmMode           string `json:"gm_mode"`
	ParticipantCount int    `json:"participant_count"`
	CharacterCount   int    `json:"character_count"`
	GmFear           int    `json:"gm_fear"`
	ThemePrompt      string `json:"theme_prompt"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// CampaignListPayload represents the MCP resource payload for campaign listings.
type CampaignListPayload struct {
	Campaigns []CampaignListEntry `json:"campaigns"`
}

// CampaignPayload represents the MCP resource payload for a single campaign.
type CampaignPayload struct {
	Campaign CampaignListEntry `json:"campaign"`
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

// CharacterListEntry represents a readable character entry.
type CharacterListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Notes      string `json:"notes"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CharacterListPayload represents the MCP resource payload for character listings.
type CharacterListPayload struct {
	Characters []CharacterListEntry `json:"characters"`
}

// CharacterCreateInput represents the MCP tool input for character creation.
type CharacterCreateInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the character"`
	Kind       string `json:"kind" jsonschema:"character kind (PC, NPC)"`
	Notes      string `json:"notes,omitempty" jsonschema:"optional free-form notes about the character"`
}

// CharacterCreateResult represents the MCP tool output for character creation.
type CharacterCreateResult struct {
	ID         string `json:"id" jsonschema:"character identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the character"`
	Kind       string `json:"kind" jsonschema:"character kind"`
	Notes      string `json:"notes" jsonschema:"free-form notes about the character"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when character was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when character was last updated"`
}

// CharacterControlSetInput represents the MCP tool input for setting character control.
type CharacterControlSetInput struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Controller  string `json:"controller" jsonschema:"controller: 'GM' (case-insensitive) for GM control, or a participant ID for participant control"`
}

// CharacterControlSetResult represents the MCP tool output for setting character control.
type CharacterControlSetResult struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Controller  string `json:"controller" jsonschema:"controller: 'GM' or the participant ID"`
}

// CharacterSheetGetInput represents the MCP tool input for getting a character sheet.
type CharacterSheetGetInput struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
}

// CharacterSheetGetResult represents the MCP tool output for getting a character sheet.
type CharacterSheetGetResult struct {
	Character CharacterCreateResult  `json:"character" jsonschema:"character metadata"`
	Profile   CharacterProfileResult `json:"profile" jsonschema:"character profile"`
	State     CharacterStateResult   `json:"state" jsonschema:"character state"`
}

// CharacterProfileResult represents character profile data in MCP responses.
type CharacterProfileResult struct {
	CharacterID     string         `json:"character_id" jsonschema:"character identifier"`
	Traits          map[string]int `json:"traits" jsonschema:"trait values"`
	HpMax           int            `json:"hp_max" jsonschema:"maximum hit points"`
	StressMax       int            `json:"stress_max" jsonschema:"maximum stress"`
	Evasion         int            `json:"evasion" jsonschema:"evasion difficulty"`
	MajorThreshold  int            `json:"major_threshold" jsonschema:"major damage threshold"`
	SevereThreshold int            `json:"severe_threshold" jsonschema:"severe damage threshold"`
}

// CharacterStateResult represents character state data in MCP responses.
type CharacterStateResult struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Hope        int    `json:"hope" jsonschema:"hope value (0..6)"`
	Stress      int    `json:"stress" jsonschema:"current stress"`
	Hp          int    `json:"hp" jsonschema:"current hit points"`
}

// CharacterProfilePatchInput represents the MCP tool input for patching a character profile.
type CharacterProfilePatchInput struct {
	CharacterID     string         `json:"character_id" jsonschema:"character identifier"`
	Traits          map[string]int `json:"traits,omitempty" jsonschema:"optional traits map (replaces entire map if provided)"`
	HpMax           *int           `json:"hp_max,omitempty" jsonschema:"optional hp_max"`
	StressMax       *int           `json:"stress_max,omitempty" jsonschema:"optional stress_max"`
	Evasion         *int           `json:"evasion,omitempty" jsonschema:"optional evasion"`
	MajorThreshold  *int           `json:"major_threshold,omitempty" jsonschema:"optional major_threshold"`
	SevereThreshold *int           `json:"severe_threshold,omitempty" jsonschema:"optional severe_threshold"`
}

// CharacterProfilePatchResult represents the MCP tool output for patching a character profile.
type CharacterProfilePatchResult struct {
	Profile CharacterProfileResult `json:"profile" jsonschema:"updated character profile"`
}

// CharacterStatePatchInput represents the MCP tool input for patching a character state.
type CharacterStatePatchInput struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Hope        *int   `json:"hope,omitempty" jsonschema:"optional hope (0..6)"`
	Stress      *int   `json:"stress,omitempty" jsonschema:"optional stress"`
	Hp          *int   `json:"hp,omitempty" jsonschema:"optional hp"`
}

// CharacterStatePatchResult represents the MCP tool output for patching a character state.
type CharacterStatePatchResult struct {
	State CharacterStateResult `json:"state" jsonschema:"updated character state"`
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

// CharacterCreateTool defines the MCP tool schema for creating characters.
func CharacterCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_create",
		Description: "Creates a character (PC or NPC) for a campaign",
	}
}

// CharacterControlSetTool defines the MCP tool schema for setting character control.
func CharacterControlSetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_control_set",
		Description: "Sets the default controller (GM or participant) for a character in a campaign",
	}
}

// CharacterSheetGetTool defines the MCP tool schema for getting a character sheet.
func CharacterSheetGetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_sheet_get",
		Description: "Gets a character sheet (character, profile, and state)",
	}
}

// CharacterProfilePatchTool defines the MCP tool schema for patching a character profile.
func CharacterProfilePatchTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_profile_patch",
		Description: "Patches a character profile (all fields optional)",
	}
}

// CharacterStatePatchTool defines the MCP tool schema for patching a character state.
func CharacterStatePatchTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_state_patch",
		Description: "Patches a character state (all fields optional)",
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

// ParticipantListResourceTemplate defines the MCP resource template for participant listings.
func ParticipantListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "participant_list",
		Title:       "Participants",
		Description: "Readable listing of participants for a campaign. URI format: campaign://{campaign_id}/participants",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/participants",
	}
}

// CharacterListResourceTemplate defines the MCP resource template for character listings.
func CharacterListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "character_list",
		Title:       "Characters",
		Description: "Readable listing of characters for a campaign. URI format: campaign://{campaign_id}/characters",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/characters",
	}
}

// CampaignResourceTemplate defines the MCP resource template for a single campaign.
func CampaignResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign",
		Title:       "Campaign",
		Description: "Readable campaign metadata record. URI format: campaign://{campaign_id}",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}",
	}
}

// CampaignCreateHandler executes a campaign creation request.
func CampaignCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[CampaignCreateInput, CampaignCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignCreateInput) (*mcp.CallToolResult, CampaignCreateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.CreateCampaign(callCtx, &campaignv1.CreateCampaignRequest{
			Name:        input.Name,
			GmMode:      gmModeFromString(input.GmMode),
			ThemePrompt: input.ThemePrompt,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response is missing")
		}

		result := CampaignCreateResult{
			ID:               response.Campaign.GetId(),
			Name:             response.Campaign.GetName(),
			GmMode:           gmModeToString(response.Campaign.GetGmMode()),
			ParticipantCount: int(response.Campaign.GetParticipantCount()),
			CharacterCount:   int(response.Campaign.GetCharacterCount()),
			GmFear:           int(response.Campaign.GetGmFear()),
			ThemePrompt:      response.Campaign.GetThemePrompt(),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		// TODO: Emit MCP notifications when campaigns are created or updated so SSE clients can refresh resources.
		return CallToolResultWithMetadata(responseMeta), result, nil
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

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := CampaignListPayload{}
		// TODO: Support page_size/page_token inputs and return next_page_token.
		response, err := client.ListCampaigns(callCtx, &campaignv1.ListCampaignsRequest{
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
				ID:               campaign.GetId(),
				Name:             campaign.GetName(),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				GmFear:           int(campaign.GetGmFear()),
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
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
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &campaignv1.CreateParticipantRequest{
			CampaignId:  input.CampaignID,
			DisplayName: input.DisplayName,
			Role:        participantRoleFromString(input.Role),
		}

		// Controller is optional; only set if provided
		if input.Controller != "" {
			req.Controller = controllerFromString(input.Controller)
		}

		response, err := client.CreateParticipant(callCtx, req, grpc.Header(&header))
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

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
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

// CharacterCreateHandler executes a character creation request.
func CharacterCreateHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[CharacterCreateInput, CharacterCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterCreateInput) (*mcp.CallToolResult, CharacterCreateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &campaignv1.CreateCharacterRequest{
			CampaignId: input.CampaignID,
			Name:       input.Name,
			Kind:       characterKindFromString(input.Kind),
			Notes:      input.Notes,
		}

		response, err := client.CreateCharacter(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("character create failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("character create response is missing")
		}

		result := CharacterCreateResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

func characterKindFromString(value string) campaignv1.CharacterKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return campaignv1.CharacterKind_PC
	case "NPC":
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

func characterKindToString(kind campaignv1.CharacterKind) string {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return "PC"
	case campaignv1.CharacterKind_NPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

// CharacterControlSetHandler executes a character control set request.
func CharacterControlSetHandler(client campaignv1.CampaignServiceClient) mcp.ToolHandlerFor[CharacterControlSetInput, CharacterControlSetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterControlSetInput) (*mcp.CallToolResult, CharacterControlSetResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		controller, err := characterControllerFromString(input.Controller)
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("invalid controller: %w", err)
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &campaignv1.SetDefaultControlRequest{
			CampaignId:  input.CampaignID,
			CharacterId: input.CharacterID,
			Controller:  controller,
		}

		response, err := client.SetDefaultControl(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set failed: %w", err)
		}
		if response == nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set response is missing")
		}

		result := CharacterControlSetResult{
			CampaignID:  response.GetCampaignId(),
			CharacterID: response.GetCharacterId(),
			Controller:  characterControllerToString(response.GetController()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// characterControllerFromString converts a string to a protobuf CharacterController.
// Accepts "GM" (case-insensitive) for GM control, or a participant ID for participant control.
func characterControllerFromString(controller string) (*campaignv1.CharacterController, error) {
	controller = strings.TrimSpace(controller)
	if controller == "" {
		return nil, fmt.Errorf("controller is required")
	}

	upper := strings.ToUpper(controller)
	if upper == "GM" {
		return &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		}, nil
	}

	// Otherwise, treat as participant ID
	return &campaignv1.CharacterController{
		Controller: &campaignv1.CharacterController_Participant{
			Participant: &campaignv1.ParticipantController{
				ParticipantId: controller,
			},
		},
	}, nil
}

// characterControllerToString converts a protobuf CharacterController to a string representation.
// Returns "GM" for GM control, or the participant ID for participant control.
func characterControllerToString(controller *campaignv1.CharacterController) string {
	if controller == nil {
		return ""
	}

	switch c := controller.GetController().(type) {
	case *campaignv1.CharacterController_Gm:
		return "GM"
	case *campaignv1.CharacterController_Participant:
		if c.Participant != nil {
			return c.Participant.GetParticipantId()
		}
		return ""
	default:
		return ""
	}
}

// CharacterSheetGetHandler executes a character sheet get request.
func CharacterSheetGetHandler(client campaignv1.CampaignServiceClient, getContext func() Context) mcp.ToolHandlerFor[CharacterSheetGetInput, CharacterSheetGetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterSheetGetInput) (*mcp.CallToolResult, CharacterSheetGetResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := getContext()
		campaignID := mcpCtx.CampaignID
		if campaignID == "" {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.GetCharacterSheet(callCtx, &campaignv1.GetCharacterSheetRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet get failed: %w", err)
		}
		if response == nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet response is missing")
		}

		traits := make(map[string]int)
		if response.Profile != nil {
			for k, v := range response.Profile.GetTraits() {
				traits[k] = int(v)
			}
		}

		result := CharacterSheetGetResult{
			Character: CharacterCreateResult{
				ID:         response.Character.GetId(),
				CampaignID: response.Character.GetCampaignId(),
				Name:       response.Character.GetName(),
				Kind:       characterKindToString(response.Character.GetKind()),
				Notes:      response.Character.GetNotes(),
				CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
			},
			Profile: CharacterProfileResult{
				CharacterID:     response.Profile.GetCharacterId(),
				Traits:          traits,
				HpMax:           int(response.Profile.GetHpMax()),
				StressMax:       int(response.Profile.GetStressMax()),
				Evasion:         int(response.Profile.GetEvasion()),
				MajorThreshold:  int(response.Profile.GetMajorThreshold()),
				SevereThreshold: int(response.Profile.GetSevereThreshold()),
			},
			State: CharacterStateResult{
				CharacterID: response.State.GetCharacterId(),
				Hope:        int(response.State.GetHope()),
				Stress:      int(response.State.GetStress()),
				Hp:          int(response.State.GetHp()),
			},
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterProfilePatchHandler executes a character profile patch request.
func CharacterProfilePatchHandler(client campaignv1.CampaignServiceClient, getContext func() Context) mcp.ToolHandlerFor[CharacterProfilePatchInput, CharacterProfilePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterProfilePatchInput) (*mcp.CallToolResult, CharacterProfilePatchResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := getContext()
		campaignID := mcpCtx.CampaignID
		if campaignID == "" {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &campaignv1.PatchCharacterProfileRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		if input.Traits != nil {
			traits := make(map[string]int32)
			for k, v := range input.Traits {
				traits[k] = int32(v)
			}
			req.Traits = traits
		}
		if input.HpMax != nil {
			hpMax := int32(*input.HpMax)
			req.HpMax = &hpMax
		}
		if input.StressMax != nil {
			stressMax := int32(*input.StressMax)
			req.StressMax = &stressMax
		}
		if input.Evasion != nil {
			evasion := int32(*input.Evasion)
			req.Evasion = &evasion
		}
		if input.MajorThreshold != nil {
			majorThreshold := int32(*input.MajorThreshold)
			req.MajorThreshold = &majorThreshold
		}
		if input.SevereThreshold != nil {
			severeThreshold := int32(*input.SevereThreshold)
			req.SevereThreshold = &severeThreshold
		}

		response, err := client.PatchCharacterProfile(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch failed: %w", err)
		}
		if response == nil || response.Profile == nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch response is missing")
		}

		traits := make(map[string]int)
		for k, v := range response.Profile.GetTraits() {
			traits[k] = int(v)
		}

		result := CharacterProfilePatchResult{
			Profile: CharacterProfileResult{
				CharacterID:     response.Profile.GetCharacterId(),
				Traits:          traits,
				HpMax:           int(response.Profile.GetHpMax()),
				StressMax:       int(response.Profile.GetStressMax()),
				Evasion:         int(response.Profile.GetEvasion()),
				MajorThreshold:  int(response.Profile.GetMajorThreshold()),
				SevereThreshold: int(response.Profile.GetSevereThreshold()),
			},
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterStatePatchHandler executes a character state patch request.
func CharacterStatePatchHandler(client campaignv1.CampaignServiceClient, getContext func() Context) mcp.ToolHandlerFor[CharacterStatePatchInput, CharacterStatePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterStatePatchInput) (*mcp.CallToolResult, CharacterStatePatchResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := getContext()
		campaignID := mcpCtx.CampaignID
		if campaignID == "" {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &campaignv1.PatchCharacterStateRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		if input.Hope != nil {
			hope := int32(*input.Hope)
			req.Hope = &hope
		}
		if input.Stress != nil {
			stress := int32(*input.Stress)
			req.Stress = &stress
		}
		if input.Hp != nil {
			hp := int32(*input.Hp)
			req.Hp = &hp
		}

		response, err := client.PatchCharacterState(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch failed: %w", err)
		}
		if response == nil || response.State == nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch response is missing")
		}

		result := CharacterStatePatchResult{
			State: CharacterStateResult{
				CharacterID: response.State.GetCharacterId(),
				Hope:        int(response.State.GetHope()),
				Stress:      int(response.State.GetStress()),
				Hp:          int(response.State.GetHp()),
			},
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantListResourceHandler returns a readable participant listing resource.
func ParticipantListResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("participant list client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/participants")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/participants.
		campaignID, err := parseCampaignIDFromURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := ParticipantListPayload{}
		response, err := client.ListParticipants(callCtx, &campaignv1.ListParticipantsRequest{
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
// It parses URIs of the expected format but requires an actual campaign ID.
func parseCampaignIDFromURI(uri string) (string, error) {
	return parseCampaignIDFromResourceURI(uri, "participants")
}

// CharacterListResourceHandler returns a readable character listing resource.
func CharacterListResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("character list client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/characters")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/characters.
		campaignID, err := parseCampaignIDFromCharacterURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := CharacterListPayload{}
		response, err := client.ListCharacters(callCtx, &campaignv1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("character list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("character list response is missing")
		}

		for _, character := range response.GetCharacters() {
			payload.Characters = append(payload.Characters, CharacterListEntry{
				ID:         character.GetId(),
				CampaignID: character.GetCampaignId(),
				Name:       character.GetName(),
				Kind:       characterKindToString(character.GetKind()),
				Notes:      character.GetNotes(),
				CreatedAt:  formatTimestamp(character.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(character.GetUpdatedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal character list: %w", err)
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

// parseCampaignIDFromCharacterURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/characters.
// It parses URIs of the expected format but requires an actual campaign ID.
func parseCampaignIDFromCharacterURI(uri string) (string, error) {
	return parseCampaignIDFromResourceURI(uri, "characters")
}

// parseCampaignIDFromCampaignURI extracts the campaign ID from a URI of the form campaign://{campaign_id}.
// It parses URIs of the expected format but requires an actual campaign ID.
// It also rejects URIs with additional path segments, query parameters, or fragments (e.g., campaign://id/participants).
func parseCampaignIDFromCampaignURI(uri string) (string, error) {
	prefix := "campaign://"

	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("URI must start with %q", prefix)
	}

	campaignID := strings.TrimPrefix(uri, prefix)
	campaignID = strings.TrimSpace(campaignID)

	if campaignID == "" {
		return "", fmt.Errorf("campaign ID is required in URI")
	}

	// Reject the placeholder value - actual campaign IDs must be provided
	if campaignID == "_" {
		return "", fmt.Errorf("campaign ID placeholder '_' is not a valid campaign ID")
	}

	// Reject URIs with additional path segments, query parameters, or fragments
	// These should be handled by other resource handlers (e.g., campaign://id/participants)
	if strings.ContainsAny(campaignID, "/?#") {
		return "", fmt.Errorf("URI must not contain path segments, query parameters, or fragments after campaign ID")
	}

	return campaignID, nil
}

// CampaignResourceHandler returns a readable single campaign resource.
func CampaignResourceHandler(client campaignv1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}.
		campaignID, err := parseCampaignIDFromCampaignURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		response, err := client.GetCampaign(callCtx, &campaignv1.GetCampaignRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			if s, ok := status.FromError(err); ok {
				if s.Code() == codes.NotFound {
					return nil, fmt.Errorf("campaign not found")
				}
				if s.Code() == codes.InvalidArgument {
					return nil, fmt.Errorf("invalid campaign_id: %s", s.Message())
				}
			}
			return nil, fmt.Errorf("get campaign failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, fmt.Errorf("campaign response is missing")
		}

		campaign := response.Campaign
		payload := CampaignPayload{
			Campaign: CampaignListEntry{
				ID:               campaign.GetId(),
				Name:             campaign.GetName(),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				GmFear:           int(campaign.GetGmFear()),
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
			},
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign: %w", err)
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
