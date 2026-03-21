package gametools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/memorydoc"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// --- Input types ---

type memorySectionReadInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Heading    string `json:"heading"`
}

type memorySectionUpdateInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Heading    string `json:"heading"`
	Content    string `json:"content"`
}

// --- Result types ---

type memorySectionResult struct {
	CampaignID string `json:"campaign_id"`
	Heading    string `json:"heading"`
	Content    string `json:"content,omitempty"`
	Found      bool   `json:"found"`
}

// --- Handlers ---

func (s *DirectSession) memorySectionRead(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input memorySectionReadInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if s.clients.Artifact == nil {
		return orchestration.ToolResult{}, fmt.Errorf("artifact manager is not configured")
	}

	record, err := s.clients.Artifact.GetArtifact(ctx, campaignID, campaigncontext.MemoryArtifactPath)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("get memory artifact: %w", err)
	}
	body, found := memorydoc.SectionRead(record.Content, input.Heading)
	return toolResultJSON(memorySectionResult{
		CampaignID: campaignID,
		Heading:    input.Heading,
		Content:    body,
		Found:      found,
	})
}

func (s *DirectSession) memorySectionUpdate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input memorySectionUpdateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if s.clients.Artifact == nil {
		return orchestration.ToolResult{}, fmt.Errorf("artifact manager is not configured")
	}

	record, err := s.clients.Artifact.GetArtifact(ctx, campaignID, campaigncontext.MemoryArtifactPath)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("get memory artifact: %w", err)
	}

	merged := memorydoc.SectionUpdate(record.Content, input.Heading, input.Content)

	if _, err := s.clients.Artifact.UpsertArtifact(ctx, campaignID, campaigncontext.MemoryArtifactPath, merged); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("upsert memory artifact: %w", err)
	}

	return toolResultJSON(memorySectionResult{
		CampaignID: campaignID,
		Heading:    input.Heading,
		Content:    input.Content,
		Found:      true,
	})
}
