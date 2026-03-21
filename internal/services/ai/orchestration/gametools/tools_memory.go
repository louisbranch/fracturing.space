package gametools

import (
	"context"
	"encoding/json"
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Artifact.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       campaigncontext.MemoryArtifactPath,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("get memory artifact: %w", err)
	}
	body, found := memorydoc.SectionRead(resp.GetArtifact().GetContent(), input.Heading)
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Artifact.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       campaigncontext.MemoryArtifactPath,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("get memory artifact: %w", err)
	}

	merged := memorydoc.SectionUpdate(resp.GetArtifact().GetContent(), input.Heading, input.Content)

	if _, err := s.clients.Artifact.UpsertCampaignArtifact(callCtx, &aiv1.UpsertCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       campaigncontext.MemoryArtifactPath,
		Content:    merged,
	}); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("upsert memory artifact: %w", err)
	}

	return toolResultJSON(memorySectionResult{
		CampaignID: campaignID,
		Heading:    input.Heading,
		Content:    input.Content,
		Found:      true,
	})
}
