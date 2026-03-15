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
		leftRank := campaignInviteStatusRank(normalized[i].Status)
		rightRank := campaignInviteStatusRank(normalized[j].Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftID := strings.TrimSpace(normalized[i].ID)
		rightID := strings.TrimSpace(normalized[j].ID)
		return leftID < rightID
	})

	return normalized, nil
}

// campaignInviteStatusRank keeps invite list ordering stable across transports
// so terminal invite states do not crowd out still-actionable pending invites.
func campaignInviteStatusRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "invite_status_pending":
		return 0
	case "claimed", "accepted", "invite_status_claimed":
		return 1
	case "declined", "rejected", "invite_status_declined":
		return 2
	case "revoked", "invite_status_revoked":
		return 3
	default:
		return 4
	}
}
