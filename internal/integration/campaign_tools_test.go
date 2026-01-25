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
}
