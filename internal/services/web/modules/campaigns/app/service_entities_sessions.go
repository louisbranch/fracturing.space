package app

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// campaignSessions centralizes this web behavior in one helper seam.
func (s service) campaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignSession{}, nil
	}

	sessions, err := s.readGateway.CampaignSessions(ctx, campaignID)
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
		if sessionName == "" {
			if sessionID != "" {
				sessionName = sessionID
			} else {
				sessionName = "Unnamed session"
			}
		}
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
func (s service) campaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (CampaignSessionReadiness, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignSessionReadiness{
			Ready:    true,
			Blockers: []CampaignSessionReadinessBlocker{},
		}, nil
	}

	readiness, err := s.readGateway.CampaignSessionReadiness(ctx, campaignID, locale)
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

// campaignInvites centralizes this web behavior in one helper seam.
func (s service) campaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignInvite{}, nil
	}

	invites, err := s.readGateway.CampaignInvites(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(invites) == 0 {
		return []CampaignInvite{}, nil
	}

	normalized := make([]CampaignInvite, 0, len(invites))
	for _, invite := range invites {
		status := strings.TrimSpace(invite.Status)
		if status == "" {
			status = "Unspecified"
		}
		normalized = append(normalized, CampaignInvite{
			ID:                strings.TrimSpace(invite.ID),
			ParticipantID:     strings.TrimSpace(invite.ParticipantID),
			ParticipantName:   strings.TrimSpace(invite.ParticipantName),
			RecipientUserID:   strings.TrimSpace(invite.RecipientUserID),
			RecipientUsername: strings.TrimSpace(invite.RecipientUsername),
			HasRecipient:      invite.HasRecipient || strings.TrimSpace(invite.RecipientUserID) != "",
			Status:            status,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		leftID := strings.TrimSpace(normalized[i].ID)
		rightID := strings.TrimSpace(normalized[j].ID)
		return leftID < rightID
	})

	return normalized, nil
}
