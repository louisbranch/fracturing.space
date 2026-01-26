//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runCampaignToolsTests exercises campaign-related MCP tools.
func runCampaignToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("campaign create and list", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		params := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Stormbound",
				"gm_mode":      "HUMAN",
				"theme_prompt": "sea and thunder",
			},
		}
		result, err := suite.client.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if result == nil {
			t.Fatal("call campaign_create returned nil")
		}
		if result.IsError {
			t.Fatalf("campaign_create returned error content: %+v", result.Content)
		}

		output := decodeStructuredContent[domain.CampaignCreateResult](t, result.StructuredContent)
		if output.ID == "" {
			t.Fatal("campaign_create returned empty id")
		}
		if output.Name != "Stormbound" {
			t.Fatalf("expected campaign name Stormbound, got %q", output.Name)
		}
		if output.GmMode != "HUMAN" {
			t.Fatalf("expected gm_mode HUMAN, got %q", output.GmMode)
		}

		resource, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: "campaigns://list"})
		if err != nil {
			t.Fatalf("read campaigns://list: %v", err)
		}
		if resource == nil || len(resource.Contents) == 0 {
			t.Fatalf("read campaigns://list returned no contents: %+v", resource)
		}

		payload := parseCampaignListPayload(t, resource.Contents[0].Text)
		entry, found := findCampaignByID(payload, output.ID)
		if !found {
			t.Fatalf("campaign %q not found in list", output.ID)
		}
		if entry.Name != output.Name {
			t.Fatalf("expected campaign name %q, got %q", output.Name, entry.Name)
		}
		if entry.GmMode != output.GmMode {
			t.Fatalf("expected gm_mode %q, got %q", output.GmMode, entry.GmMode)
		}
		createdAt := parseRFC3339(t, entry.CreatedAt)
		updatedAt := parseRFC3339(t, entry.UpdatedAt)
		if updatedAt.Before(createdAt) {
			t.Fatalf("expected updated_at after created_at: %v < %v", updatedAt, createdAt)
		}
	})

	t.Run("participant create", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// First create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Test Campaign",
				"gm_mode":      "HUMAN",
				"theme_prompt": "",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)

		// Now create a participant
		participantParams := &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Test Player",
				"role":         "PLAYER",
				"controller":   "HUMAN",
			},
		}
		participantResult, err := suite.client.CallTool(ctx, participantParams)
		if err != nil {
			t.Fatalf("call participant_create: %v", err)
		}
		if participantResult == nil {
			t.Fatal("call participant_create returned nil")
		}
		if participantResult.IsError {
			t.Fatalf("participant_create returned error content: %+v", participantResult.Content)
		}

		output := decodeStructuredContent[domain.ParticipantCreateResult](t, participantResult.StructuredContent)
		if output.ID == "" {
			t.Fatal("participant_create returned empty id")
		}
		if output.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign id %q, got %q", campaignOutput.ID, output.CampaignID)
		}
		if output.DisplayName != "Test Player" {
			t.Fatalf("expected display name Test Player, got %q", output.DisplayName)
		}
		if output.Role != "PLAYER" {
			t.Fatalf("expected role PLAYER, got %q", output.Role)
		}
		if output.Controller != "HUMAN" {
			t.Fatalf("expected controller HUMAN, got %q", output.Controller)
		}
		if output.CreatedAt == "" {
			t.Fatal("participant_create returned empty created_at")
		}
		if output.UpdatedAt == "" {
			t.Fatal("participant_create returned empty updated_at")
		}
		createdAt := parseRFC3339(t, output.CreatedAt)
		updatedAt := parseRFC3339(t, output.UpdatedAt)
		if updatedAt.Before(createdAt) {
			t.Fatalf("expected updated_at after created_at: %v < %v", updatedAt, createdAt)
		}
	})

	t.Run("character create", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// First create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Test Campaign",
				"gm_mode":      "HUMAN",
				"theme_prompt": "",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)

		// Test creating a PC character
		characterParams := &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test PC",
				"kind":        "PC",
				"notes":       "A brave warrior",
			},
		}
		characterResult, err := suite.client.CallTool(ctx, characterParams)
		if err != nil {
			t.Fatalf("call character_create: %v", err)
		}
		if characterResult == nil {
			t.Fatal("call character_create returned nil")
		}
		if characterResult.IsError {
			t.Fatalf("character_create returned error content: %+v", characterResult.Content)
		}

		output := decodeStructuredContent[domain.CharacterCreateResult](t, characterResult.StructuredContent)
		if output.ID == "" {
			t.Fatal("character_create returned empty id")
		}
		if output.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign id %q, got %q", campaignOutput.ID, output.CampaignID)
		}
		if output.Name != "Test PC" {
			t.Fatalf("expected name Test PC, got %q", output.Name)
		}
		if output.Kind != "PC" {
			t.Fatalf("expected kind PC, got %q", output.Kind)
		}
		if output.Notes != "A brave warrior" {
			t.Fatalf("expected notes A brave warrior, got %q", output.Notes)
		}
		if output.CreatedAt == "" {
			t.Fatal("character_create returned empty created_at")
		}
		if output.UpdatedAt == "" {
			t.Fatal("character_create returned empty updated_at")
		}
		createdAt := parseRFC3339(t, output.CreatedAt)
		updatedAt := parseRFC3339(t, output.UpdatedAt)
		if updatedAt.Before(createdAt) {
			t.Fatalf("expected updated_at after created_at: %v < %v", updatedAt, createdAt)
		}

		// Test creating an NPC character with optional notes omitted
		npcParams := &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test NPC",
				"kind":        "NPC",
			},
		}
		npcResult, err := suite.client.CallTool(ctx, npcParams)
		if err != nil {
			t.Fatalf("call character_create for NPC: %v", err)
		}
		if npcResult == nil || npcResult.IsError {
			t.Fatalf("character_create for NPC failed: %+v", npcResult)
		}
		npcOutput := decodeStructuredContent[domain.CharacterCreateResult](t, npcResult.StructuredContent)
		if npcOutput.ID == "" {
			t.Fatal("character_create for NPC returned empty id")
		}
		if npcOutput.Kind != "NPC" {
			t.Fatalf("expected kind NPC, got %q", npcOutput.Kind)
		}
		if npcOutput.Name != "Test NPC" {
			t.Fatalf("expected name Test NPC, got %q", npcOutput.Name)
		}
	})

	t.Run("character control set", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// First create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Test Campaign",
				"gm_mode":      "HUMAN",
				"theme_prompt": "",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)

		// Create a character
		characterParams := &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test Character",
				"kind":        "PC",
			},
		}
		characterResult, err := suite.client.CallTool(ctx, characterParams)
		if err != nil {
			t.Fatalf("call character_create: %v", err)
		}
		if characterResult == nil || characterResult.IsError {
			t.Fatalf("character_create failed: %+v", characterResult)
		}
		characterOutput := decodeStructuredContent[domain.CharacterCreateResult](t, characterResult.StructuredContent)

		// Test setting GM controller
		gmControlParams := &mcp.CallToolParams{
			Name: "character_control_set",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"character_id": characterOutput.ID,
				"controller":   "GM",
			},
		}
		gmControlResult, err := suite.client.CallTool(ctx, gmControlParams)
		if err != nil {
			t.Fatalf("call character_control_set with GM: %v", err)
		}
		if gmControlResult == nil {
			t.Fatal("call character_control_set returned nil")
		}
		if gmControlResult.IsError {
			t.Fatalf("character_control_set returned error content: %+v", gmControlResult.Content)
		}

		gmControlOutput := decodeStructuredContent[domain.CharacterControlSetResult](t, gmControlResult.StructuredContent)
		if gmControlOutput.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign id %q, got %q", campaignOutput.ID, gmControlOutput.CampaignID)
		}
		if gmControlOutput.CharacterID != characterOutput.ID {
			t.Fatalf("expected character id %q, got %q", characterOutput.ID, gmControlOutput.CharacterID)
		}
		if gmControlOutput.Controller != "GM" {
			t.Fatalf("expected controller GM, got %q", gmControlOutput.Controller)
		}

		// Create a participant for participant controller test
		participantParams := &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Test Player",
				"role":         "PLAYER",
			},
		}
		participantResult, err := suite.client.CallTool(ctx, participantParams)
		if err != nil {
			t.Fatalf("call participant_create: %v", err)
		}
		if participantResult == nil || participantResult.IsError {
			t.Fatalf("participant_create failed: %+v", participantResult)
		}
		participantOutput := decodeStructuredContent[domain.ParticipantCreateResult](t, participantResult.StructuredContent)

		// Test setting participant controller
		participantControlParams := &mcp.CallToolParams{
			Name: "character_control_set",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"character_id": characterOutput.ID,
				"controller":   participantOutput.ID,
			},
		}
		participantControlResult, err := suite.client.CallTool(ctx, participantControlParams)
		if err != nil {
			t.Fatalf("call character_control_set with participant: %v", err)
		}
		if participantControlResult == nil {
			t.Fatal("call character_control_set returned nil")
		}
		if participantControlResult.IsError {
			t.Fatalf("character_control_set returned error content: %+v", participantControlResult.Content)
		}

		participantControlOutput := decodeStructuredContent[domain.CharacterControlSetResult](t, participantControlResult.StructuredContent)
		if participantControlOutput.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign id %q, got %q", campaignOutput.ID, participantControlOutput.CampaignID)
		}
		if participantControlOutput.CharacterID != characterOutput.ID {
			t.Fatalf("expected character id %q, got %q", characterOutput.ID, participantControlOutput.CharacterID)
		}
		if participantControlOutput.Controller != participantOutput.ID {
			t.Fatalf("expected controller %q, got %q", participantOutput.ID, participantControlOutput.Controller)
		}
	})
}
