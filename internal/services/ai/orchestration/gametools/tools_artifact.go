package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Artifact.ListCampaignArtifacts(callCtx, &aiv1.ListCampaignArtifactsRequest{CampaignId: campaignID})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact list failed: %w", err)
	}
	result := artifactListResult{CampaignID: campaignID}
	for _, a := range resp.GetArtifacts() {
		result.Artifacts = append(result.Artifacts, artifactFromProto(a, false))
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Artifact.GetCampaignArtifact(callCtx, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       input.Path,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact get failed: %w", err)
	}
	return toolResultJSON(artifactFromProto(resp.GetArtifact(), true))
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
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Artifact.UpsertCampaignArtifact(callCtx, &aiv1.UpsertCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       input.Path,
		Content:    input.Content,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("campaign artifact upsert failed: %w", err)
	}
	return toolResultJSON(artifactFromProto(resp.GetArtifact(), true))
}

func artifactFromProto(a *aiv1.CampaignArtifact, includeContent bool) artifactResult {
	if a == nil {
		return artifactResult{}
	}
	r := artifactResult{
		CampaignID: a.GetCampaignId(),
		Path:       a.GetPath(),
		ReadOnly:   a.GetReadOnly(),
		CreatedAt:  formatArtifactTimestamp(a.GetCreatedAt()),
		UpdatedAt:  formatArtifactTimestamp(a.GetUpdatedAt()),
	}
	if includeContent {
		r.Content = a.GetContent()
	}
	return r
}

func formatArtifactTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}
