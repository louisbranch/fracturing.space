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

// CampaignGameSurface centralizes this web behavior in one helper seam.
func (s gameService) CampaignGameSurface(ctx context.Context, campaignID string) (CampaignGameSurface, error) {
	return s.campaignGameSurface(ctx, campaignID)
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

// campaignGameSurface centralizes this web behavior in one helper seam.
func (s gameService) campaignGameSurface(ctx context.Context, campaignID string) (CampaignGameSurface, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignGameSurface{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	surface, err := s.read.CampaignGameSurface(ctx, campaignID)
	if err != nil {
		return CampaignGameSurface{}, err
	}
	surface.Participant.ID = strings.TrimSpace(surface.Participant.ID)
	surface.Participant.Name = strings.TrimSpace(surface.Participant.Name)
	if surface.Participant.Name == "" {
		surface.Participant.Name = surface.Participant.ID
	}
	surface.Participant.Role = strings.TrimSpace(surface.Participant.Role)
	if surface.Participant.Role == "" {
		surface.Participant.Role = "Unspecified"
	}
	surface.SessionID = strings.TrimSpace(surface.SessionID)
	surface.SessionName = strings.TrimSpace(surface.SessionName)
	if surface.SessionName == "" {
		surface.SessionName = surface.SessionID
	}
	surface.DefaultStreamID = strings.TrimSpace(surface.DefaultStreamID)
	surface.DefaultPersonaID = strings.TrimSpace(surface.DefaultPersonaID)
	if len(surface.Streams) == 0 {
		surface.Streams = []CampaignGameStream{}
	}
	for i := range surface.Streams {
		surface.Streams[i].ID = strings.TrimSpace(surface.Streams[i].ID)
		surface.Streams[i].Kind = strings.TrimSpace(surface.Streams[i].Kind)
		surface.Streams[i].Scope = strings.TrimSpace(surface.Streams[i].Scope)
		surface.Streams[i].SessionID = strings.TrimSpace(surface.Streams[i].SessionID)
		surface.Streams[i].SceneID = strings.TrimSpace(surface.Streams[i].SceneID)
		surface.Streams[i].Label = strings.TrimSpace(surface.Streams[i].Label)
		if surface.Streams[i].Label == "" {
			surface.Streams[i].Label = surface.Streams[i].ID
		}
		if surface.DefaultStreamID == "" && surface.Streams[i].ID != "" {
			surface.DefaultStreamID = surface.Streams[i].ID
		}
	}
	if len(surface.Personas) == 0 {
		surface.Personas = []CampaignGamePersona{}
	}
	for i := range surface.Personas {
		surface.Personas[i].ID = strings.TrimSpace(surface.Personas[i].ID)
		surface.Personas[i].Kind = strings.TrimSpace(surface.Personas[i].Kind)
		surface.Personas[i].ParticipantID = strings.TrimSpace(surface.Personas[i].ParticipantID)
		surface.Personas[i].CharacterID = strings.TrimSpace(surface.Personas[i].CharacterID)
		surface.Personas[i].DisplayName = strings.TrimSpace(surface.Personas[i].DisplayName)
		if surface.Personas[i].DisplayName == "" {
			surface.Personas[i].DisplayName = surface.Personas[i].ID
		}
		if surface.DefaultPersonaID == "" && surface.Personas[i].ID != "" {
			surface.DefaultPersonaID = surface.Personas[i].ID
		}
	}
	if surface.ActiveSessionGate != nil {
		surface.ActiveSessionGate.ID = strings.TrimSpace(surface.ActiveSessionGate.ID)
		surface.ActiveSessionGate.Type = strings.TrimSpace(surface.ActiveSessionGate.Type)
		surface.ActiveSessionGate.Status = strings.TrimSpace(surface.ActiveSessionGate.Status)
		surface.ActiveSessionGate.Reason = strings.TrimSpace(surface.ActiveSessionGate.Reason)
		if len(surface.ActiveSessionGate.Metadata) == 0 {
			surface.ActiveSessionGate.Metadata = nil
		}
	}
	if surface.ActiveSessionSpotlight != nil {
		surface.ActiveSessionSpotlight.Type = strings.TrimSpace(surface.ActiveSessionSpotlight.Type)
		surface.ActiveSessionSpotlight.CharacterID = strings.TrimSpace(surface.ActiveSessionSpotlight.CharacterID)
	}
	return surface, nil
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
