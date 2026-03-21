package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// --- Input types ---

type artifactListInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
}

type artifactGetInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Path       string `json:"path"`
}

type artifactUpsertInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Path       string `json:"path"`
	Content    string `json:"content"`
}

// --- Result types ---

type artifactResult struct {
	CampaignID string `json:"campaign_id"`
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	ReadOnly   bool   `json:"read_only"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type artifactListResult struct {
	CampaignID string           `json:"campaign_id"`
	Artifacts  []artifactResult `json:"artifacts"`
}

// --- Handlers ---

func (s *DirectSession) artifactList(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input artifactListInput
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

	records, err := s.clients.Artifact.ListArtifacts(ctx, campaignID)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact list failed: %w", err)
	}
	result := artifactListResult{CampaignID: campaignID}
	for _, r := range records {
		result.Artifacts = append(result.Artifacts, artifactFromRecord(r, false))
	}
	return toolResultJSON(result)
}

func (s *DirectSession) artifactGet(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input artifactGetInput
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

	record, err := s.clients.Artifact.GetArtifact(ctx, campaignID, input.Path)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact get failed: %w", err)
	}
	return toolResultJSON(artifactFromRecord(record, true))
}

func (s *DirectSession) artifactUpsert(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input artifactUpsertInput
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

	record, err := s.clients.Artifact.UpsertArtifact(ctx, campaignID, input.Path, input.Content)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact upsert failed: %w", err)
	}
	return toolResultJSON(artifactFromRecord(record, true))
}

func artifactFromRecord(r storage.CampaignArtifactRecord, includeContent bool) artifactResult {
	result := artifactResult{
		CampaignID: r.CampaignID,
		Path:       r.Path,
		ReadOnly:   r.ReadOnly,
		CreatedAt:  formatArtifactTime(r.CreatedAt),
		UpdatedAt:  formatArtifactTime(r.UpdatedAt),
	}
	if includeContent {
		result.Content = r.Content
	}
	return result
}

func formatArtifactTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
