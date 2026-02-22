package domain

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
