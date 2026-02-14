package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CampaignCreateInput represents the MCP tool input for campaign creation.
type CampaignCreateInput struct {
	Name         string `json:"name" jsonschema:"campaign name"`
	System       string `json:"system" jsonschema:"game system (DAGGERHEART)"`
	GmMode       string `json:"gm_mode" jsonschema:"gm mode (HUMAN, AI, HYBRID)"`
	Intent       string `json:"intent,omitempty" jsonschema:"campaign intent (STANDARD, STARTER, SANDBOX)"`
	AccessPolicy string `json:"access_policy,omitempty" jsonschema:"campaign access policy (PRIVATE, RESTRICTED, PUBLIC)"`
	ThemePrompt  string `json:"theme_prompt,omitempty" jsonschema:"optional theme prompt"`
	UserID       string `json:"user_id,omitempty" jsonschema:"creator user identifier"`
}

// CampaignStatusChangeInput represents the MCP tool input for campaign lifecycle changes.
type CampaignStatusChangeInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
}

// CampaignCreateResult represents the MCP tool output for campaign creation.
type CampaignCreateResult struct {
	ID                 string `json:"id" jsonschema:"campaign identifier"`
	OwnerParticipantID string `json:"owner_participant_id" jsonschema:"owner participant identifier for setting context"`
	Name               string `json:"name" jsonschema:"campaign name"`
	GmMode             string `json:"gm_mode" jsonschema:"gm mode"`
	Intent             string `json:"intent" jsonschema:"campaign intent"`
	AccessPolicy       string `json:"access_policy" jsonschema:"campaign access policy"`
	ParticipantCount   int    `json:"participant_count" jsonschema:"number of all participants (GM + PLAYER + future roles)"`
	CharacterCount     int    `json:"character_count" jsonschema:"number of all characters (PC + NPC + future kinds)"`
	GmFear             int    `json:"gm_fear" jsonschema:"campaign-scoped GM fear"`
	ThemePrompt        string `json:"theme_prompt" jsonschema:"theme prompt"`
	Status             string `json:"status" jsonschema:"campaign status"`
	CreatedAt          string `json:"created_at" jsonschema:"RFC3339 timestamp when campaign was created"`
	UpdatedAt          string `json:"updated_at" jsonschema:"RFC3339 timestamp when campaign was last updated"`
	CompletedAt        string `json:"completed_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was completed"`
	ArchivedAt         string `json:"archived_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was archived"`
}

// CampaignStatusResult represents the MCP tool output for campaign lifecycle changes.
type CampaignStatusResult struct {
	ID               string `json:"id" jsonschema:"campaign identifier"`
	Name             string `json:"name" jsonschema:"campaign name"`
	GmMode           string `json:"gm_mode" jsonschema:"gm mode"`
	Intent           string `json:"intent" jsonschema:"campaign intent"`
	AccessPolicy     string `json:"access_policy" jsonschema:"campaign access policy"`
	ParticipantCount int    `json:"participant_count" jsonschema:"number of all participants (GM + PLAYER + future roles)"`
	CharacterCount   int    `json:"character_count" jsonschema:"number of all characters (PC + NPC + future kinds)"`
	GmFear           int    `json:"gm_fear" jsonschema:"campaign-scoped GM fear"`
	ThemePrompt      string `json:"theme_prompt" jsonschema:"theme prompt"`
	Status           string `json:"status" jsonschema:"campaign status"`
	CreatedAt        string `json:"created_at" jsonschema:"RFC3339 timestamp when campaign was created"`
	UpdatedAt        string `json:"updated_at" jsonschema:"RFC3339 timestamp when campaign was last updated"`
	CompletedAt      string `json:"completed_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was completed"`
	ArchivedAt       string `json:"archived_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was archived"`
}

// CampaignListEntry represents a readable campaign metadata entry.
type CampaignListEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	GmMode           string `json:"gm_mode"`
	Intent           string `json:"intent"`
	AccessPolicy     string `json:"access_policy"`
	ParticipantCount int    `json:"participant_count"`
	CharacterCount   int    `json:"character_count"`
	GmFear           int    `json:"gm_fear"`
	ThemePrompt      string `json:"theme_prompt"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
	ArchivedAt       string `json:"archived_at,omitempty"`
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

// ParticipantUpdateInput represents the MCP tool input for participant updates.
type ParticipantUpdateInput struct {
	CampaignID    string  `json:"campaign_id" jsonschema:"campaign identifier"`
	ParticipantID string  `json:"participant_id" jsonschema:"participant identifier"`
	DisplayName   *string `json:"display_name,omitempty" jsonschema:"optional display name"`
	Role          *string `json:"role,omitempty" jsonschema:"optional participant role (GM, PLAYER)"`
	Controller    *string `json:"controller,omitempty" jsonschema:"optional controller (HUMAN, AI)"`
}

// ParticipantDeleteInput represents the MCP tool input for participant deletion.
type ParticipantDeleteInput struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant identifier"`
	Reason        string `json:"reason,omitempty" jsonschema:"optional reason for deletion"`
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

// ParticipantUpdateResult represents the MCP tool output for participant updates.
type ParticipantUpdateResult struct {
	ID          string `json:"id" jsonschema:"participant identifier"`
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	DisplayName string `json:"display_name" jsonschema:"display name for the participant"`
	Role        string `json:"role" jsonschema:"participant role"`
	Controller  string `json:"controller" jsonschema:"controller type"`
	CreatedAt   string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt   string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// ParticipantDeleteResult represents the MCP tool output for participant deletion.
type ParticipantDeleteResult struct {
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

// CharacterUpdateInput represents the MCP tool input for character updates.
type CharacterUpdateInput struct {
	CampaignID  string  `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string  `json:"character_id" jsonschema:"character identifier"`
	Name        *string `json:"name,omitempty" jsonschema:"optional display name for the character"`
	Kind        *string `json:"kind,omitempty" jsonschema:"optional character kind (PC, NPC)"`
	Notes       *string `json:"notes,omitempty" jsonschema:"optional free-form notes about the character"`
}

// CharacterUpdateResult represents the MCP tool output for character updates.
type CharacterUpdateResult struct {
	ID         string `json:"id" jsonschema:"character identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the character"`
	Kind       string `json:"kind" jsonschema:"character kind"`
	Notes      string `json:"notes" jsonschema:"free-form notes about the character"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when character was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when character was last updated"`
}

// CharacterDeleteInput represents the MCP tool input for character deletion.
type CharacterDeleteInput struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Reason      string `json:"reason,omitempty" jsonschema:"optional reason for deletion"`
}

// CharacterDeleteResult represents the MCP tool output for character deletion.
type CharacterDeleteResult struct {
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
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID   string `json:"character_id" jsonschema:"character identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant id to control the character (empty to unassign)"`
}

// CharacterControlSetResult represents the MCP tool output for setting character control.
type CharacterControlSetResult struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID   string `json:"character_id" jsonschema:"character identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant id assigned to the character"`
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
	CharacterID     string `json:"character_id" jsonschema:"character identifier"`
	HpMax           int    `json:"hp_max" jsonschema:"maximum hit points"`
	StressMax       int    `json:"stress_max" jsonschema:"maximum stress"`
	Evasion         int    `json:"evasion" jsonschema:"evasion difficulty"`
	MajorThreshold  int    `json:"major_threshold" jsonschema:"major damage threshold"`
	SevereThreshold int    `json:"severe_threshold" jsonschema:"severe damage threshold"`
	// Daggerheart traits
	Agility   int `json:"agility" jsonschema:"agility trait (-2 to +4)"`
	Strength  int `json:"strength" jsonschema:"strength trait (-2 to +4)"`
	Finesse   int `json:"finesse" jsonschema:"finesse trait (-2 to +4)"`
	Instinct  int `json:"instinct" jsonschema:"instinct trait (-2 to +4)"`
	Presence  int `json:"presence" jsonschema:"presence trait (-2 to +4)"`
	Knowledge int `json:"knowledge" jsonschema:"knowledge trait (-2 to +4)"`
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
	CharacterID     string `json:"character_id" jsonschema:"character identifier"`
	HpMax           *int   `json:"hp_max,omitempty" jsonschema:"optional hp_max"`
	StressMax       *int   `json:"stress_max,omitempty" jsonschema:"optional stress_max"`
	Evasion         *int   `json:"evasion,omitempty" jsonschema:"optional evasion"`
	MajorThreshold  *int   `json:"major_threshold,omitempty" jsonschema:"optional major_threshold"`
	SevereThreshold *int   `json:"severe_threshold,omitempty" jsonschema:"optional severe_threshold"`
	// Daggerheart traits (optional, -2 to +4)
	Agility   *int `json:"agility,omitempty" jsonschema:"optional agility trait"`
	Strength  *int `json:"strength,omitempty" jsonschema:"optional strength trait"`
	Finesse   *int `json:"finesse,omitempty" jsonschema:"optional finesse trait"`
	Instinct  *int `json:"instinct,omitempty" jsonschema:"optional instinct trait"`
	Presence  *int `json:"presence,omitempty" jsonschema:"optional presence trait"`
	Knowledge *int `json:"knowledge,omitempty" jsonschema:"optional knowledge trait"`
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

// characterProfileResultFromProto converts a proto CharacterProfile to MCP result type.
// Extracts Daggerheart-specific fields from the oneof extension.
func characterProfileResultFromProto(profile *statev1.CharacterProfile) CharacterProfileResult {
	result := CharacterProfileResult{
		CharacterID: profile.GetCharacterId(),
	}

	// Extract Daggerheart-specific fields if present (includes HP max)
	if dh := profile.GetDaggerheart(); dh != nil {
		result.HpMax = int(dh.GetHpMax())
		result.StressMax = int(dh.GetStressMax().GetValue())
		result.Evasion = int(dh.GetEvasion().GetValue())
		result.MajorThreshold = int(dh.GetMajorThreshold().GetValue())
		result.SevereThreshold = int(dh.GetSevereThreshold().GetValue())
		result.Agility = int(dh.GetAgility().GetValue())
		result.Strength = int(dh.GetStrength().GetValue())
		result.Finesse = int(dh.GetFinesse().GetValue())
		result.Instinct = int(dh.GetInstinct().GetValue())
		result.Presence = int(dh.GetPresence().GetValue())
		result.Knowledge = int(dh.GetKnowledge().GetValue())
	}

	return result
}

// characterStateResultFromProto converts a proto CharacterState to MCP result type.
// Extracts Daggerheart-specific fields from the oneof extension.
func characterStateResultFromProto(state *statev1.CharacterState) CharacterStateResult {
	result := CharacterStateResult{
		CharacterID: state.GetCharacterId(),
	}

	// Extract Daggerheart-specific fields if present (includes HP)
	if dh := state.GetDaggerheart(); dh != nil {
		result.Hp = int(dh.GetHp())
		result.Hope = int(dh.GetHope())
		result.Stress = int(dh.GetStress())
	}

	return result
}

// CampaignCreateTool defines the MCP tool schema for creating campaigns.
func CampaignCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_create",
		Description: "Creates a new campaign metadata record",
	}
}

// CampaignEndTool defines the MCP tool schema for ending campaigns.
func CampaignEndTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_end",
		Description: "Marks a campaign as completed",
	}
}

// CampaignArchiveTool defines the MCP tool schema for archiving campaigns.
func CampaignArchiveTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_archive",
		Description: "Archives a campaign",
	}
}

// CampaignRestoreTool defines the MCP tool schema for restoring campaigns.
func CampaignRestoreTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_restore",
		Description: "Restores an archived campaign to draft",
	}
}

// ParticipantCreateTool defines the MCP tool schema for creating participants.
func ParticipantCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "participant_create",
		Description: "Creates a participant (GM or player) for a campaign",
	}
}

// ParticipantUpdateTool defines the MCP tool schema for updating participants.
func ParticipantUpdateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "participant_update",
		Description: "Updates a participant's metadata (display name, role, controller)",
	}
}

// ParticipantDeleteTool defines the MCP tool schema for deleting participants.
func ParticipantDeleteTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "participant_delete",
		Description: "Deletes a participant from a campaign",
	}
}

// CharacterCreateTool defines the MCP tool schema for creating characters.
func CharacterCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_create",
		Description: "Creates a character (PC or NPC) for a campaign",
	}
}

// CharacterUpdateTool defines the MCP tool schema for updating characters.
func CharacterUpdateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_update",
		Description: "Updates a character's metadata (name, kind, notes)",
	}
}

// CharacterDeleteTool defines the MCP tool schema for deleting characters.
func CharacterDeleteTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_delete",
		Description: "Deletes a character from a campaign",
	}
}

// CharacterControlSetTool defines the MCP tool schema for setting character control.
func CharacterControlSetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "character_control_set",
		Description: "Sets the participant controller for a character in a campaign",
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
func CampaignCreateHandler(client statev1.CampaignServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignCreateInput, CampaignCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignCreateInput) (*mcp.CallToolResult, CampaignCreateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		if strings.TrimSpace(input.UserID) != "" {
			callCtx = metadata.AppendToOutgoingContext(callCtx, grpcmeta.UserIDHeader, strings.TrimSpace(input.UserID))
		}

		var header metadata.MD

		response, err := client.CreateCampaign(callCtx, &statev1.CreateCampaignRequest{
			Name:         input.Name,
			System:       gameSystemFromString(input.System),
			GmMode:       gmModeFromString(input.GmMode),
			Intent:       campaignIntentFromString(input.Intent),
			AccessPolicy: campaignAccessPolicyFromString(input.AccessPolicy),
			ThemePrompt:  input.ThemePrompt,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response is missing")
		}
		if response.OwnerParticipant == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response missing owner participant")
		}
		ownerParticipantID := response.OwnerParticipant.GetId()
		if ownerParticipantID == "" {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response missing owner participant id")
		}

		result := CampaignCreateResult{
			ID:                 response.Campaign.GetId(),
			OwnerParticipantID: ownerParticipantID,
			Name:               response.Campaign.GetName(),
			GmMode:             gmModeToString(response.Campaign.GetGmMode()),
			Intent:             campaignIntentToString(response.Campaign.GetIntent()),
			AccessPolicy:       campaignAccessPolicyToString(response.Campaign.GetAccessPolicy()),
			ParticipantCount:   int(response.Campaign.GetParticipantCount()),
			CharacterCount:     int(response.Campaign.GetCharacterCount()),
			GmFear:             0, // GM Fear is now in Snapshot, not Campaign
			ThemePrompt:        response.Campaign.GetThemePrompt(),
			Status:             campaignStatusToString(response.Campaign.GetStatus()),
			CreatedAt:          formatTimestamp(response.Campaign.GetCreatedAt()),
			UpdatedAt:          formatTimestamp(response.Campaign.GetUpdatedAt()),
			CompletedAt:        formatTimestamp(response.Campaign.GetCompletedAt()),
			ArchivedAt:         formatTimestamp(response.Campaign.GetArchivedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignEndHandler executes a campaign end request.
func CampaignEndHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}
		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = mcpCtx.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.EndCampaign(callCtx, &statev1.EndCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign end failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign end response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignArchiveHandler executes a campaign archive request.
func CampaignArchiveHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}
		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = mcpCtx.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.ArchiveCampaign(callCtx, &statev1.ArchiveCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign archive failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign archive response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignRestoreHandler executes a campaign restore request.
func CampaignRestoreHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}
		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = mcpCtx.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.RestoreCampaign(callCtx, &statev1.RestoreCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign restore failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign restore response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignListResourceHandler returns a readable campaign listing resource.
func CampaignListResourceHandler(client statev1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign list client is not configured")
		}

		uri := CampaignListResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := CampaignListPayload{}
		// TODO: Support page_size/page_token inputs and return next_page_token.
		response, err := client.ListCampaigns(callCtx, &statev1.ListCampaignsRequest{
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
				Status:           campaignStatusToString(campaign.GetStatus()),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				Intent:           campaignIntentToString(campaign.GetIntent()),
				AccessPolicy:     campaignAccessPolicyToString(campaign.GetAccessPolicy()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				GmFear:           0, // GM Fear is now in Snapshot, not Campaign
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
				CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
				ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
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

func gameSystemFromString(value string) commonv1.GameSystem {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAGGERHEART", "GAME_SYSTEM_DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

func gmModeFromString(value string) statev1.GmMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return statev1.GmMode_HUMAN
	case "AI":
		return statev1.GmMode_AI
	case "HYBRID":
		return statev1.GmMode_HYBRID
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func gmModeToString(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "HUMAN"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

func campaignIntentFromString(value string) statev1.CampaignIntent {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "STANDARD", "CAMPAIGN_INTENT_STANDARD":
		return statev1.CampaignIntent_STANDARD
	case "STARTER", "CAMPAIGN_INTENT_STARTER":
		return statev1.CampaignIntent_STARTER
	case "SANDBOX", "CAMPAIGN_INTENT_SANDBOX":
		return statev1.CampaignIntent_SANDBOX
	default:
		return statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED
	}
}

func campaignIntentToString(intent statev1.CampaignIntent) string {
	switch intent {
	case statev1.CampaignIntent_STANDARD:
		return "STANDARD"
	case statev1.CampaignIntent_STARTER:
		return "STARTER"
	case statev1.CampaignIntent_SANDBOX:
		return "SANDBOX"
	default:
		return "UNSPECIFIED"
	}
}

func campaignAccessPolicyFromString(value string) statev1.CampaignAccessPolicy {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PRIVATE", "CAMPAIGN_ACCESS_POLICY_PRIVATE":
		return statev1.CampaignAccessPolicy_PRIVATE
	case "RESTRICTED", "CAMPAIGN_ACCESS_POLICY_RESTRICTED":
		return statev1.CampaignAccessPolicy_RESTRICTED
	case "PUBLIC", "CAMPAIGN_ACCESS_POLICY_PUBLIC":
		return statev1.CampaignAccessPolicy_PUBLIC
	default:
		return statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED
	}
}

func campaignAccessPolicyToString(policy statev1.CampaignAccessPolicy) string {
	switch policy {
	case statev1.CampaignAccessPolicy_PRIVATE:
		return "PRIVATE"
	case statev1.CampaignAccessPolicy_RESTRICTED:
		return "RESTRICTED"
	case statev1.CampaignAccessPolicy_PUBLIC:
		return "PUBLIC"
	default:
		return "UNSPECIFIED"
	}
}

func campaignStatusToString(status statev1.CampaignStatus) string {
	switch status {
	case statev1.CampaignStatus_DRAFT:
		return "DRAFT"
	case statev1.CampaignStatus_ACTIVE:
		return "ACTIVE"
	case statev1.CampaignStatus_COMPLETED:
		return "COMPLETED"
	case statev1.CampaignStatus_ARCHIVED:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

func campaignStatusResultFromProto(campaign *statev1.Campaign) CampaignStatusResult {
	return CampaignStatusResult{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		GmMode:           gmModeToString(campaign.GetGmMode()),
		Intent:           campaignIntentToString(campaign.GetIntent()),
		AccessPolicy:     campaignAccessPolicyToString(campaign.GetAccessPolicy()),
		ParticipantCount: int(campaign.GetParticipantCount()),
		CharacterCount:   int(campaign.GetCharacterCount()),
		GmFear:           0, // GM Fear is now in Snapshot, not Campaign
		ThemePrompt:      campaign.GetThemePrompt(),
		Status:           campaignStatusToString(campaign.GetStatus()),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
		CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
		ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
	}
}

// ParticipantCreateHandler executes a participant creation request.
func ParticipantCreateHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantCreateInput, ParticipantCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantCreateInput) (*mcp.CallToolResult, ParticipantCreateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(runCtx, invocationID, mcpCtx)
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.CreateParticipantRequest{
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
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantUpdateHandler executes a participant update request.
func ParticipantUpdateHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantUpdateInput, ParticipantUpdateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantUpdateInput) (*mcp.CallToolResult, ParticipantUpdateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(runCtx, invocationID, mcpCtx)
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.ParticipantID == "" {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant_id is required")
		}

		req := &statev1.UpdateParticipantRequest{
			CampaignId:    input.CampaignID,
			ParticipantId: input.ParticipantID,
		}
		if input.DisplayName != nil {
			req.DisplayName = wrapperspb.String(*input.DisplayName)
		}
		if input.Role != nil {
			role := participantRoleFromString(*input.Role)
			if role == statev1.ParticipantRole_ROLE_UNSPECIFIED {
				return nil, ParticipantUpdateResult{}, fmt.Errorf("role must be GM or PLAYER")
			}
			req.Role = role
		}
		if input.Controller != nil {
			controller := controllerFromString(*input.Controller)
			if controller == statev1.Controller_CONTROLLER_UNSPECIFIED {
				return nil, ParticipantUpdateResult{}, fmt.Errorf("controller must be HUMAN or AI")
			}
			req.Controller = controller
		}
		if req.DisplayName == nil && req.Role == statev1.ParticipantRole_ROLE_UNSPECIFIED && req.Controller == statev1.Controller_CONTROLLER_UNSPECIFIED {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("at least one field must be provided")
		}

		var header metadata.MD
		response, err := client.UpdateParticipant(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant update failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant update response is missing")
		}

		result := ParticipantUpdateResult{
			ID:          response.Participant.GetId(),
			CampaignID:  response.Participant.GetCampaignId(),
			DisplayName: response.Participant.GetDisplayName(),
			Role:        participantRoleToString(response.Participant.GetRole()),
			Controller:  controllerToString(response.Participant.GetController()),
			CreatedAt:   formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:   formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantDeleteHandler executes a participant delete request.
func ParticipantDeleteHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantDeleteInput, ParticipantDeleteResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantDeleteInput) (*mcp.CallToolResult, ParticipantDeleteResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(runCtx, invocationID, mcpCtx)
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.ParticipantID == "" {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant_id is required")
		}

		var header metadata.MD
		response, err := client.DeleteParticipant(callCtx, &statev1.DeleteParticipantRequest{
			CampaignId:    input.CampaignID,
			ParticipantId: input.ParticipantID,
			Reason:        input.Reason,
		}, grpc.Header(&header))
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant delete failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant delete response is missing")
		}

		result := ParticipantDeleteResult{
			ID:          response.Participant.GetId(),
			CampaignID:  response.Participant.GetCampaignId(),
			DisplayName: response.Participant.GetDisplayName(),
			Role:        participantRoleToString(response.Participant.GetRole()),
			Controller:  controllerToString(response.Participant.GetController()),
			CreatedAt:   formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:   formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

func participantRoleFromString(value string) statev1.ParticipantRole {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GM":
		return statev1.ParticipantRole_GM
	case "PLAYER":
		return statev1.ParticipantRole_PLAYER
	default:
		return statev1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

func participantRoleToString(role statev1.ParticipantRole) string {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM"
	case statev1.ParticipantRole_PLAYER:
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

func controllerFromString(value string) statev1.Controller {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return statev1.Controller_CONTROLLER_HUMAN
	case "AI":
		return statev1.Controller_CONTROLLER_AI
	default:
		return statev1.Controller_CONTROLLER_UNSPECIFIED
	}
}

func controllerToString(controller statev1.Controller) string {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "HUMAN"
	case statev1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}

// CharacterCreateHandler executes a character creation request.
func CharacterCreateHandler(client statev1.CharacterServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterCreateInput, CharacterCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterCreateInput) (*mcp.CallToolResult, CharacterCreateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.CreateCharacterRequest{
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
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterUpdateHandler executes a character update request.
func CharacterUpdateHandler(client statev1.CharacterServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterUpdateInput, CharacterUpdateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterUpdateInput) (*mcp.CallToolResult, CharacterUpdateResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, CharacterUpdateResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.CharacterID == "" {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character_id is required")
		}

		req := &statev1.UpdateCharacterRequest{
			CampaignId:  input.CampaignID,
			CharacterId: input.CharacterID,
		}
		if input.Name != nil {
			req.Name = wrapperspb.String(*input.Name)
		}
		if input.Kind != nil {
			kind := characterKindFromString(*input.Kind)
			if kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
				return nil, CharacterUpdateResult{}, fmt.Errorf("kind must be PC or NPC")
			}
			req.Kind = kind
		}
		if input.Notes != nil {
			req.Notes = wrapperspb.String(*input.Notes)
		}
		if req.Name == nil && req.Kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED && req.Notes == nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("at least one field must be provided")
		}

		var header metadata.MD
		response, err := client.UpdateCharacter(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character update failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character update response is missing")
		}

		result := CharacterUpdateResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterDeleteHandler executes a character delete request.
func CharacterDeleteHandler(client statev1.CharacterServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterDeleteInput, CharacterDeleteResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterDeleteInput) (*mcp.CallToolResult, CharacterDeleteResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, CharacterDeleteResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.CharacterID == "" {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character_id is required")
		}

		var header metadata.MD
		response, err := client.DeleteCharacter(callCtx, &statev1.DeleteCharacterRequest{
			CampaignId:  input.CampaignID,
			CharacterId: input.CharacterID,
			Reason:      input.Reason,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character delete failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character delete response is missing")
		}

		result := CharacterDeleteResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

func characterKindFromString(value string) statev1.CharacterKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return statev1.CharacterKind_PC
	case "NPC":
		return statev1.CharacterKind_NPC
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

func characterKindToString(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

// CharacterControlSetHandler executes a character control set request.
func CharacterControlSetHandler(client statev1.CharacterServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterControlSetInput, CharacterControlSetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterControlSetInput) (*mcp.CallToolResult, CharacterControlSetResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.SetDefaultControlRequest{
			CampaignId:    input.CampaignID,
			CharacterId:   input.CharacterID,
			ParticipantId: wrapperspb.String(input.ParticipantID),
		}

		response, err := client.SetDefaultControl(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set failed: %w", err)
		}
		if response == nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set response is missing")
		}

		participantID := ""
		if response.GetParticipantId() != nil {
			participantID = response.GetParticipantId().GetValue()
		}
		result := CharacterControlSetResult{
			CampaignID:    response.GetCampaignId(),
			CharacterID:   response.GetCharacterId(),
			ParticipantID: participantID,
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterSheetGetHandler executes a character sheet get request.
func CharacterSheetGetHandler(client statev1.CharacterServiceClient, getContext func() Context) mcp.ToolHandlerFor[CharacterSheetGetInput, CharacterSheetGetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterSheetGetInput) (*mcp.CallToolResult, CharacterSheetGetResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
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

		response, err := client.GetCharacterSheet(callCtx, &statev1.GetCharacterSheetRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet get failed: %w", err)
		}
		if response == nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet response is missing")
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
			Profile: characterProfileResultFromProto(response.Profile),
			State:   characterStateResultFromProto(response.State),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterProfilePatchHandler executes a character profile patch request.
func CharacterProfilePatchHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterProfilePatchInput, CharacterProfilePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterProfilePatchInput) (*mcp.CallToolResult, CharacterProfilePatchResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
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

		req := &statev1.PatchCharacterProfileRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		// All profile fields are now Daggerheart-specific (including hp_max)
		hasDaggerheartPatch := input.HpMax != nil || input.StressMax != nil || input.Evasion != nil ||
			input.MajorThreshold != nil || input.SevereThreshold != nil ||
			input.Agility != nil || input.Strength != nil || input.Finesse != nil ||
			input.Instinct != nil || input.Presence != nil || input.Knowledge != nil
		if hasDaggerheartPatch {
			dhProfile := &daggerheartv1.DaggerheartProfile{}
			if input.HpMax != nil {
				dhProfile.HpMax = int32(*input.HpMax)
			}
			if input.StressMax != nil {
				dhProfile.StressMax = wrapperspb.Int32(int32(*input.StressMax))
			}
			if input.Evasion != nil {
				dhProfile.Evasion = wrapperspb.Int32(int32(*input.Evasion))
			}
			if input.MajorThreshold != nil {
				dhProfile.MajorThreshold = wrapperspb.Int32(int32(*input.MajorThreshold))
			}
			if input.SevereThreshold != nil {
				dhProfile.SevereThreshold = wrapperspb.Int32(int32(*input.SevereThreshold))
			}
			if input.Agility != nil {
				dhProfile.Agility = wrapperspb.Int32(int32(*input.Agility))
			}
			if input.Strength != nil {
				dhProfile.Strength = wrapperspb.Int32(int32(*input.Strength))
			}
			if input.Finesse != nil {
				dhProfile.Finesse = wrapperspb.Int32(int32(*input.Finesse))
			}
			if input.Instinct != nil {
				dhProfile.Instinct = wrapperspb.Int32(int32(*input.Instinct))
			}
			if input.Presence != nil {
				dhProfile.Presence = wrapperspb.Int32(int32(*input.Presence))
			}
			if input.Knowledge != nil {
				dhProfile.Knowledge = wrapperspb.Int32(int32(*input.Knowledge))
			}
			req.SystemProfilePatch = &statev1.PatchCharacterProfileRequest_Daggerheart{
				Daggerheart: dhProfile,
			}
		}

		response, err := client.PatchCharacterProfile(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch failed: %w", err)
		}
		if response == nil || response.Profile == nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch response is missing")
		}

		result := CharacterProfilePatchResult{
			Profile: characterProfileResultFromProto(response.Profile),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", campaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterStatePatchHandler executes a character state patch request.
func CharacterStatePatchHandler(client statev1.SnapshotServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterStatePatchInput, CharacterStatePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterStatePatchInput) (*mcp.CallToolResult, CharacterStatePatchResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
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

		req := &statev1.PatchCharacterStateRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		// All state fields are now Daggerheart-specific (including HP)
		if input.Hp != nil || input.Hope != nil || input.Stress != nil {
			dhState := &daggerheartv1.DaggerheartCharacterState{}
			if input.Hp != nil {
				dhState.Hp = int32(*input.Hp)
			}
			if input.Hope != nil {
				dhState.Hope = int32(*input.Hope)
			}
			if input.Stress != nil {
				dhState.Stress = int32(*input.Stress)
			}
			req.SystemStatePatch = &statev1.PatchCharacterStateRequest_Daggerheart{
				Daggerheart: dhState,
			}
		}

		response, err := client.PatchCharacterState(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch failed: %w", err)
		}
		if response == nil || response.State == nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch response is missing")
		}

		result := CharacterStatePatchResult{
			State: characterStateResultFromProto(response.State),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", campaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantListResourceHandler returns a readable participant listing resource.
func ParticipantListResourceHandler(client statev1.ParticipantServiceClient) mcp.ResourceHandler {
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

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := ParticipantListPayload{}
		response, err := client.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
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
func CharacterListResourceHandler(client statev1.CharacterServiceClient) mcp.ResourceHandler {
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

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := CharacterListPayload{}
		response, err := client.ListCharacters(callCtx, &statev1.ListCharactersRequest{
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
func CampaignResourceHandler(client statev1.CampaignServiceClient) mcp.ResourceHandler {
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

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		response, err := client.GetCampaign(callCtx, &statev1.GetCampaignRequest{
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
				Status:           campaignStatusToString(campaign.GetStatus()),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				GmFear:           0, // GM Fear is now in Snapshot, not Campaign
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
				CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
				ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
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
