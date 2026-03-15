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
	if surface.ActiveScene != nil {
		surface.ActiveScene.ID = strings.TrimSpace(surface.ActiveScene.ID)
		surface.ActiveScene.SessionID = strings.TrimSpace(surface.ActiveScene.SessionID)
		surface.ActiveScene.Name = strings.TrimSpace(surface.ActiveScene.Name)
		if surface.ActiveScene.Name == "" {
			surface.ActiveScene.Name = surface.ActiveScene.ID
		}
		surface.ActiveScene.Description = strings.TrimSpace(surface.ActiveScene.Description)
		if len(surface.ActiveScene.Characters) == 0 {
			surface.ActiveScene.Characters = []CampaignGameCharacter{}
		}
		for i := range surface.ActiveScene.Characters {
			surface.ActiveScene.Characters[i].ID = strings.TrimSpace(surface.ActiveScene.Characters[i].ID)
			surface.ActiveScene.Characters[i].Name = strings.TrimSpace(surface.ActiveScene.Characters[i].Name)
			if surface.ActiveScene.Characters[i].Name == "" {
				surface.ActiveScene.Characters[i].Name = surface.ActiveScene.Characters[i].ID
			}
			surface.ActiveScene.Characters[i].OwnerParticipantID = strings.TrimSpace(surface.ActiveScene.Characters[i].OwnerParticipantID)
		}
	}
	if surface.PlayerPhase != nil {
		surface.PlayerPhase.PhaseID = strings.TrimSpace(surface.PlayerPhase.PhaseID)
		surface.PlayerPhase.Status = strings.TrimSpace(surface.PlayerPhase.Status)
		if surface.PlayerPhase.Status == "" {
			surface.PlayerPhase.Status = "gm"
		}
		surface.PlayerPhase.FrameText = strings.TrimSpace(surface.PlayerPhase.FrameText)
		surface.PlayerPhase.ActingCharacterIDs = trimStringSlice(surface.PlayerPhase.ActingCharacterIDs)
		surface.PlayerPhase.ActingParticipantIDs = trimStringSlice(surface.PlayerPhase.ActingParticipantIDs)
		if len(surface.PlayerPhase.Slots) == 0 {
			surface.PlayerPhase.Slots = []CampaignGamePlayerSlot{}
		}
		for i := range surface.PlayerPhase.Slots {
			surface.PlayerPhase.Slots[i].ParticipantID = strings.TrimSpace(surface.PlayerPhase.Slots[i].ParticipantID)
			surface.PlayerPhase.Slots[i].SummaryText = strings.TrimSpace(surface.PlayerPhase.Slots[i].SummaryText)
			surface.PlayerPhase.Slots[i].CharacterIDs = trimStringSlice(surface.PlayerPhase.Slots[i].CharacterIDs)
			surface.PlayerPhase.Slots[i].ReviewStatus = strings.TrimSpace(surface.PlayerPhase.Slots[i].ReviewStatus)
			surface.PlayerPhase.Slots[i].ReviewReason = strings.TrimSpace(surface.PlayerPhase.Slots[i].ReviewReason)
			surface.PlayerPhase.Slots[i].ReviewCharacterIDs = trimStringSlice(surface.PlayerPhase.Slots[i].ReviewCharacterIDs)
		}
	}
	surface.OOC.ReadyToResumeParticipantIDs = trimStringSlice(surface.OOC.ReadyToResumeParticipantIDs)
	if len(surface.OOC.Posts) == 0 {
		surface.OOC.Posts = []CampaignGameOOCPost{}
	}
	for i := range surface.OOC.Posts {
		surface.OOC.Posts[i].PostID = strings.TrimSpace(surface.OOC.Posts[i].PostID)
		surface.OOC.Posts[i].ParticipantID = strings.TrimSpace(surface.OOC.Posts[i].ParticipantID)
		surface.OOC.Posts[i].Body = strings.TrimSpace(surface.OOC.Posts[i].Body)
	}
	surface.GMAuthorityParticipantID = strings.TrimSpace(surface.GMAuthorityParticipantID)
	surface.AITurn.Status = strings.TrimSpace(surface.AITurn.Status)
	if surface.AITurn.Status == "" {
		surface.AITurn.Status = "idle"
	}
	surface.AITurn.TurnToken = strings.TrimSpace(surface.AITurn.TurnToken)
	surface.AITurn.OwnerParticipantID = strings.TrimSpace(surface.AITurn.OwnerParticipantID)
	surface.AITurn.SourceEventType = strings.TrimSpace(surface.AITurn.SourceEventType)
	surface.AITurn.SourceSceneID = strings.TrimSpace(surface.AITurn.SourceSceneID)
	surface.AITurn.SourcePhaseID = strings.TrimSpace(surface.AITurn.SourcePhaseID)
	surface.AITurn.LastError = strings.TrimSpace(surface.AITurn.LastError)
	return surface, nil
}

// trimStringSlice normalizes transport slices into compact, non-empty values.
func trimStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
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
