//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runForkToolsTests exercises campaign forking MCP tools.
func runForkToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("fork campaign at current state", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign to fork from
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Original Campaign",
				"system":       "DAGGERHEART",
				"gm_mode":      "HUMAN",
				"theme_prompt": "A test campaign for forking",
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

		// Fork the campaign at current state (no event_seq specified)
		forkParams := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": campaignOutput.ID,
				"new_campaign_name":  "Forked Campaign",
				"copy_participants":  false,
			},
		}
		forkResult, err := suite.client.CallTool(ctx, forkParams)
		if err != nil {
			t.Fatalf("call campaign_fork: %v", err)
		}
		if forkResult == nil {
			t.Fatal("call campaign_fork returned nil")
		}
		if forkResult.IsError {
			t.Fatalf("campaign_fork returned error content: %+v", forkResult.Content)
		}

		forkOutput := decodeStructuredContent[domain.CampaignForkResult](t, forkResult.StructuredContent)
		if forkOutput.CampaignID == "" {
			t.Fatal("campaign_fork returned empty campaign_id")
		}
		if forkOutput.CampaignID == campaignOutput.ID {
			t.Fatalf("forked campaign ID should differ from source: %s", forkOutput.CampaignID)
		}
		if forkOutput.Name != "Forked Campaign" {
			t.Fatalf("expected name 'Forked Campaign', got %q", forkOutput.Name)
		}
		if forkOutput.ParentCampaignID != campaignOutput.ID {
			t.Fatalf("expected parent_campaign_id %q, got %q", campaignOutput.ID, forkOutput.ParentCampaignID)
		}
		if forkOutput.Status != "DRAFT" {
			t.Fatalf("expected status DRAFT, got %q", forkOutput.Status)
		}
		if forkOutput.CreatedAt == "" {
			t.Fatal("campaign_fork returned empty created_at")
		}
		_ = parseRFC3339(t, forkOutput.CreatedAt)
	})

	t.Run("fork campaign with auto-generated name", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign to fork from
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "My Adventure",
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

		// Fork without specifying name
		forkParams := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": campaignOutput.ID,
			},
		}
		forkResult, err := suite.client.CallTool(ctx, forkParams)
		if err != nil {
			t.Fatalf("call campaign_fork: %v", err)
		}
		if forkResult == nil || forkResult.IsError {
			t.Fatalf("campaign_fork failed: %+v", forkResult)
		}

		forkOutput := decodeStructuredContent[domain.CampaignForkResult](t, forkResult.StructuredContent)
		if forkOutput.Name != "My Adventure (Fork)" {
			t.Fatalf("expected auto-generated name 'My Adventure (Fork)', got %q", forkOutput.Name)
		}
	})

	t.Run("get campaign lineage for original", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create an original campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Original Campaign",
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

		// Get lineage of original campaign
		lineageParams := &mcp.CallToolParams{
			Name: "campaign_lineage",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		lineageResult, err := suite.client.CallTool(ctx, lineageParams)
		if err != nil {
			t.Fatalf("call campaign_lineage: %v", err)
		}
		if lineageResult == nil {
			t.Fatal("call campaign_lineage returned nil")
		}
		if lineageResult.IsError {
			t.Fatalf("campaign_lineage returned error content: %+v", lineageResult.Content)
		}

		lineageOutput := decodeStructuredContent[domain.CampaignLineageResult](t, lineageResult.StructuredContent)
		if lineageOutput.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign_id %q, got %q", campaignOutput.ID, lineageOutput.CampaignID)
		}
		if lineageOutput.ParentCampaignID != "" {
			t.Fatalf("expected empty parent_campaign_id for original, got %q", lineageOutput.ParentCampaignID)
		}
		if lineageOutput.OriginCampaignID != campaignOutput.ID {
			t.Fatalf("expected origin_campaign_id to be self for original, got %q", lineageOutput.OriginCampaignID)
		}
		if lineageOutput.Depth != 0 {
			t.Fatalf("expected depth 0 for original, got %d", lineageOutput.Depth)
		}
		if !lineageOutput.IsOriginal {
			t.Fatal("expected is_original to be true for original campaign")
		}
	})

	t.Run("get campaign lineage for fork", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create an original campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Original Campaign",
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

		// Fork the campaign
		forkParams := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": campaignOutput.ID,
				"new_campaign_name":  "First Fork",
			},
		}
		forkResult, err := suite.client.CallTool(ctx, forkParams)
		if err != nil {
			t.Fatalf("call campaign_fork: %v", err)
		}
		if forkResult == nil || forkResult.IsError {
			t.Fatalf("campaign_fork failed: %+v", forkResult)
		}
		forkOutput := decodeStructuredContent[domain.CampaignForkResult](t, forkResult.StructuredContent)

		// Get lineage of forked campaign
		lineageParams := &mcp.CallToolParams{
			Name: "campaign_lineage",
			Arguments: map[string]any{
				"campaign_id": forkOutput.CampaignID,
			},
		}
		lineageResult, err := suite.client.CallTool(ctx, lineageParams)
		if err != nil {
			t.Fatalf("call campaign_lineage: %v", err)
		}
		if lineageResult == nil || lineageResult.IsError {
			t.Fatalf("campaign_lineage failed: %+v", lineageResult)
		}

		lineageOutput := decodeStructuredContent[domain.CampaignLineageResult](t, lineageResult.StructuredContent)
		if lineageOutput.CampaignID != forkOutput.CampaignID {
			t.Fatalf("expected campaign_id %q, got %q", forkOutput.CampaignID, lineageOutput.CampaignID)
		}
		if lineageOutput.ParentCampaignID != campaignOutput.ID {
			t.Fatalf("expected parent_campaign_id %q, got %q", campaignOutput.ID, lineageOutput.ParentCampaignID)
		}
		if lineageOutput.OriginCampaignID != campaignOutput.ID {
			t.Fatalf("expected origin_campaign_id %q, got %q", campaignOutput.ID, lineageOutput.OriginCampaignID)
		}
		if lineageOutput.Depth != 1 {
			t.Fatalf("expected depth 1 for first fork, got %d", lineageOutput.Depth)
		}
		if lineageOutput.IsOriginal {
			t.Fatal("expected is_original to be false for forked campaign")
		}
	})

	t.Run("fork a fork (nested forking)", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create an original campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Root Campaign",
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
		originalOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)

		// Fork the original
		fork1Params := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": originalOutput.ID,
				"new_campaign_name":  "First Fork",
			},
		}
		fork1Result, err := suite.client.CallTool(ctx, fork1Params)
		if err != nil {
			t.Fatalf("call campaign_fork (first): %v", err)
		}
		if fork1Result == nil || fork1Result.IsError {
			t.Fatalf("campaign_fork (first) failed: %+v", fork1Result)
		}
		fork1Output := decodeStructuredContent[domain.CampaignForkResult](t, fork1Result.StructuredContent)

		// Fork the fork
		fork2Params := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": fork1Output.CampaignID,
				"new_campaign_name":  "Second Fork",
			},
		}
		fork2Result, err := suite.client.CallTool(ctx, fork2Params)
		if err != nil {
			t.Fatalf("call campaign_fork (second): %v", err)
		}
		if fork2Result == nil || fork2Result.IsError {
			t.Fatalf("campaign_fork (second) failed: %+v", fork2Result)
		}
		fork2Output := decodeStructuredContent[domain.CampaignForkResult](t, fork2Result.StructuredContent)

		// Verify second fork's lineage
		if fork2Output.ParentCampaignID != fork1Output.CampaignID {
			t.Fatalf("expected parent_campaign_id %q, got %q", fork1Output.CampaignID, fork2Output.ParentCampaignID)
		}
		if fork2Output.OriginCampaignID != originalOutput.ID {
			t.Fatalf("expected origin_campaign_id %q, got %q", originalOutput.ID, fork2Output.OriginCampaignID)
		}

		// Get lineage of deeply nested fork
		lineageParams := &mcp.CallToolParams{
			Name: "campaign_lineage",
			Arguments: map[string]any{
				"campaign_id": fork2Output.CampaignID,
			},
		}
		lineageResult, err := suite.client.CallTool(ctx, lineageParams)
		if err != nil {
			t.Fatalf("call campaign_lineage: %v", err)
		}
		if lineageResult == nil || lineageResult.IsError {
			t.Fatalf("campaign_lineage failed: %+v", lineageResult)
		}

		lineageOutput := decodeStructuredContent[domain.CampaignLineageResult](t, lineageResult.StructuredContent)
		if lineageOutput.Depth != 2 {
			t.Fatalf("expected depth 2 for second fork, got %d", lineageOutput.Depth)
		}
		if lineageOutput.OriginCampaignID != originalOutput.ID {
			t.Fatalf("expected origin_campaign_id to trace back to root, got %q", lineageOutput.OriginCampaignID)
		}
	})

	t.Run("fork non-existent campaign", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		forkParams := &mcp.CallToolParams{
			Name: "campaign_fork",
			Arguments: map[string]any{
				"source_campaign_id": "non-existent-id",
			},
		}
		forkResult, err := suite.client.CallTool(ctx, forkParams)
		if err != nil {
			t.Fatalf("call campaign_fork: %v", err)
		}
		if forkResult == nil {
			t.Fatal("call campaign_fork returned nil")
		}
		if !forkResult.IsError {
			t.Fatal("expected campaign_fork to return error for non-existent campaign")
		}
	})

	t.Run("get lineage for non-existent campaign", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		lineageParams := &mcp.CallToolParams{
			Name: "campaign_lineage",
			Arguments: map[string]any{
				"campaign_id": "non-existent-id",
			},
		}
		lineageResult, err := suite.client.CallTool(ctx, lineageParams)
		if err != nil {
			t.Fatalf("call campaign_lineage: %v", err)
		}
		if lineageResult == nil {
			t.Fatal("call campaign_lineage returned nil")
		}
		if !lineageResult.IsError {
			t.Fatal("expected campaign_lineage to return error for non-existent campaign")
		}
	})
}
