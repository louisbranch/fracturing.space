package app

import (
	"context"
	"sort"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ListCampaigns returns the package view collection for this workflow.
func (s catalogService) ListCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	return s.listCampaigns(ctx)
}

// CampaignName centralizes this web behavior in one helper seam.
func (s workspaceService) CampaignName(ctx context.Context, campaignID string) string {
	return s.campaignName(ctx, campaignID)
}

// CampaignWorkspace centralizes this web behavior in one helper seam.
func (s workspaceService) CampaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	return s.campaignWorkspace(ctx, campaignID)
}

// listCampaigns returns the package view collection for this workflow.
func (s catalogService) listCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	items, err := s.read.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []CampaignSummary{}, nil
	}
	sorted := append([]CampaignSummary(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAtUnixNano > sorted[j].CreatedAtUnixNano
	})
	return sorted, nil
}

// campaignName centralizes this web behavior in one helper seam.
func (s workspaceService) campaignName(ctx context.Context, campaignID string) string {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ""
	}
	name, err := s.read.CampaignName(ctx, campaignID)
	if err != nil {
		return campaignID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return campaignID
	}
	return name
}

// campaignWorkspace centralizes this web behavior in one helper seam.
func (s workspaceService) campaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignWorkspace{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	workspace, err := loadCampaignWorkspace(ctx, s.read, campaignID)
	if err != nil {
		return CampaignWorkspace{}, err
	}
	return normalizeCampaignWorkspace(campaignID, workspace), nil
}

// loadCampaignWorkspace fetches raw workspace state from one owned read seam.
func loadCampaignWorkspace(ctx context.Context, read CampaignWorkspaceReadGateway, campaignID string) (CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignWorkspace{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	return read.CampaignWorkspace(ctx, campaignID)
}

// normalizeCampaignWorkspace applies the transport-facing workspace defaults
// shared by workspace, participant, and configuration flows.
func normalizeCampaignWorkspace(campaignID string, workspace CampaignWorkspace) CampaignWorkspace {
	campaignID = strings.TrimSpace(campaignID)
	workspace.ID = campaignID
	workspace.Name = strings.TrimSpace(workspace.Name)
	if workspace.Name == "" {
		workspace.Name = campaignID
	}
	workspace.Theme = strings.TrimSpace(workspace.Theme)
	workspace.System = strings.TrimSpace(workspace.System)
	if workspace.System == "" {
		workspace.System = "Unspecified"
	}
	workspace.GMMode = strings.TrimSpace(workspace.GMMode)
	if workspace.GMMode == "" {
		workspace.GMMode = "Unspecified"
	}
	workspace.Status = strings.TrimSpace(workspace.Status)
	if workspace.Status == "" {
		workspace.Status = "Unspecified"
	}
	workspace.Locale = strings.TrimSpace(workspace.Locale)
	if workspace.Locale == "" {
		workspace.Locale = "Unspecified"
	}
	workspace.ParticipantCount = strings.TrimSpace(workspace.ParticipantCount)
	if workspace.ParticipantCount == "" {
		workspace.ParticipantCount = "0"
	}
	workspace.CharacterCount = strings.TrimSpace(workspace.CharacterCount)
	if workspace.CharacterCount == "" {
		workspace.CharacterCount = "0"
	}
	workspace.Intent = strings.TrimSpace(workspace.Intent)
	if workspace.Intent == "" {
		workspace.Intent = "Unspecified"
	}
	workspace.AccessPolicy = strings.TrimSpace(workspace.AccessPolicy)
	if workspace.AccessPolicy == "" {
		workspace.AccessPolicy = "Unspecified"
	}
	workspace.CoverPreviewURL = strings.TrimSpace(workspace.CoverPreviewURL)
	if workspace.CoverPreviewURL == "" {
		workspace.CoverPreviewURL = CampaignCoverPreviewImageURL("", campaignID, "", "")
	}
	workspace.CoverImageURL = strings.TrimSpace(workspace.CoverImageURL)
	if workspace.CoverImageURL == "" {
		workspace.CoverImageURL = CampaignCoverBackgroundImageURL("", campaignID, "", "")
	}
	return workspace
}
