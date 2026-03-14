package app

import (
	"context"
	"sort"
	"strings"
)

// CampaignInvites centralizes this web behavior in one helper seam.
func (s inviteReadService) CampaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	return s.campaignInvites(ctx, campaignID)
}

// campaignInvites centralizes this web behavior in one helper seam.
func (s inviteReadService) campaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignInvite{}, nil
	}

	invites, err := s.read.CampaignInvites(ctx, campaignID)
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
