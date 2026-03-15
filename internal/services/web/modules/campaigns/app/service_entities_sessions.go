package app

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// CampaignSessions centralizes this web behavior in one helper seam.
func (s sessionReadService) CampaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	return s.campaignSessions(ctx, campaignID)
}

// CampaignSessionReadiness centralizes this web behavior in one helper seam.
func (s sessionReadService) CampaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (CampaignSessionReadiness, error) {
	return s.campaignSessionReadiness(ctx, campaignID, locale)
}

// campaignSessions centralizes this web behavior in one helper seam.
func (s sessionReadService) campaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignSession{}, nil
	}

	sessions, err := s.read.CampaignSessions(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return []CampaignSession{}, nil
	}

	normalized := make([]CampaignSession, 0, len(sessions))
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		sessionName := strings.TrimSpace(session.Name)
		status := strings.TrimSpace(session.Status)
		if status == "" {
			status = "Unspecified"
		}
		normalized = append(normalized, CampaignSession{
			ID:        sessionID,
			Name:      sessionName,
			Status:    status,
			StartedAt: strings.TrimSpace(session.StartedAt),
			UpdatedAt: strings.TrimSpace(session.UpdatedAt),
			EndedAt:   strings.TrimSpace(session.EndedAt),
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		leftUpdated := strings.TrimSpace(normalized[i].UpdatedAt)
		rightUpdated := strings.TrimSpace(normalized[j].UpdatedAt)
		if leftUpdated == rightUpdated {
			return strings.TrimSpace(normalized[i].ID) < strings.TrimSpace(normalized[j].ID)
		}
		return leftUpdated > rightUpdated
	})

	return normalized, nil
}

// campaignSessionReadiness centralizes this web behavior in one helper seam.
func (s sessionReadService) campaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (CampaignSessionReadiness, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignSessionReadiness{
			Ready:    true,
			Blockers: []CampaignSessionReadinessBlocker{},
		}, nil
	}

	readiness, err := s.read.CampaignSessionReadiness(ctx, campaignID, locale)
	if err != nil {
		return CampaignSessionReadiness{}, err
	}

	normalized := CampaignSessionReadiness{
		Ready:    readiness.Ready,
		Blockers: make([]CampaignSessionReadinessBlocker, 0, len(readiness.Blockers)),
	}
	for _, blocker := range readiness.Blockers {
		metadata := make(map[string]string, len(blocker.Metadata))
		for key, value := range blocker.Metadata {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			metadata[trimmedKey] = strings.TrimSpace(value)
		}
		code := strings.TrimSpace(blocker.Code)
		message := strings.TrimSpace(blocker.Message)
		if message == "" {
			message = code
		}
		normalized.Blockers = append(normalized.Blockers, CampaignSessionReadinessBlocker{
			Code:     code,
			Message:  message,
			Metadata: metadata,
		})
	}
	if normalized.Ready {
		normalized.Blockers = []CampaignSessionReadinessBlocker{}
	}
	return normalized, nil
}
