//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runCampaignToolsTests exercises campaign-related MCP tools.
func runCampaignToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("participant create", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// First create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Test Campaign",
				"system":       "DAGGERHEART",
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
				"system":       "DAGGERHEART",
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
		if npcOutput.Notes != "" {
			t.Fatalf("expected empty notes for NPC, got %q", npcOutput.Notes)
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
				"system":       "DAGGERHEART",
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
		if participantControlOutput.Controller != participantOutput.ID {
			t.Fatalf("expected controller %q, got %q", participantOutput.ID, participantControlOutput.Controller)
		}
	})

	t.Run("campaign lifecycle", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Test Campaign",
				"system":       "DAGGERHEART",
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

		startSessionParams := &mcp.CallToolParams{
			Name: "session_start",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		startSessionResult, err := suite.client.CallTool(ctx, startSessionParams)
		if err != nil {
			t.Fatalf("call session_start: %v", err)
		}
		if startSessionResult == nil || startSessionResult.IsError {
			t.Fatalf("session_start failed: %+v", startSessionResult)
		}
		sessionOutput := decodeStructuredContent[domain.SessionStartResult](t, startSessionResult.StructuredContent)

		endSessionParams := &mcp.CallToolParams{
			Name: "session_end",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"session_id":  sessionOutput.ID,
			},
		}
		endSessionResult, err := suite.client.CallTool(ctx, endSessionParams)
		if err != nil {
			t.Fatalf("call session_end: %v", err)
		}
		if endSessionResult == nil || endSessionResult.IsError {
			t.Fatalf("session_end failed: %+v", endSessionResult)
		}

		endCampaignParams := &mcp.CallToolParams{
			Name: "campaign_end",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		endCampaignResult, err := suite.client.CallTool(ctx, endCampaignParams)
		if err != nil {
			t.Fatalf("call campaign_end: %v", err)
		}
		if endCampaignResult == nil || endCampaignResult.IsError {
			t.Fatalf("campaign_end failed: %+v", endCampaignResult)
		}
		endOutput := decodeStructuredContent[domain.CampaignStatusResult](t, endCampaignResult.StructuredContent)
		if endOutput.Status != "COMPLETED" {
			t.Fatalf("expected status COMPLETED, got %q", endOutput.Status)
		}
		if endOutput.CompletedAt == "" {
			t.Fatal("campaign_end returned empty completed_at")
		}
		_ = parseRFC3339(t, endOutput.CompletedAt)

		archiveCampaignParams := &mcp.CallToolParams{
			Name: "campaign_archive",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		archiveCampaignResult, err := suite.client.CallTool(ctx, archiveCampaignParams)
		if err != nil {
			t.Fatalf("call campaign_archive: %v", err)
		}
		if archiveCampaignResult == nil || archiveCampaignResult.IsError {
			t.Fatalf("campaign_archive failed: %+v", archiveCampaignResult)
		}
		archiveOutput := decodeStructuredContent[domain.CampaignStatusResult](t, archiveCampaignResult.StructuredContent)
		if archiveOutput.Status != "ARCHIVED" {
			t.Fatalf("expected status ARCHIVED, got %q", archiveOutput.Status)
		}
		if archiveOutput.ArchivedAt == "" {
			t.Fatal("campaign_archive returned empty archived_at")
		}
		_ = parseRFC3339(t, archiveOutput.ArchivedAt)

		restoreCampaignParams := &mcp.CallToolParams{
			Name: "campaign_restore",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		restoreCampaignResult, err := suite.client.CallTool(ctx, restoreCampaignParams)
		if err != nil {
			t.Fatalf("call campaign_restore: %v", err)
		}
		if restoreCampaignResult == nil || restoreCampaignResult.IsError {
			t.Fatalf("campaign_restore failed: %+v", restoreCampaignResult)
		}
		restoreOutput := decodeStructuredContent[domain.CampaignStatusResult](t, restoreCampaignResult.StructuredContent)
		if restoreOutput.Status != "DRAFT" {
			t.Fatalf("expected status DRAFT, got %q", restoreOutput.Status)
		}
		if restoreOutput.CompletedAt != "" {
			t.Fatalf("expected completed_at cleared, got %q", restoreOutput.CompletedAt)
		}
		if restoreOutput.ArchivedAt != "" {
			t.Fatalf("expected archived_at cleared, got %q", restoreOutput.ArchivedAt)
		}
	})
}
