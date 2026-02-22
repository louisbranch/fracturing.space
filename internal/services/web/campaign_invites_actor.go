package web

import (
	"context"
	"errors"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

type campaignInviteActor struct {
	participantID    string
	canManageInvites bool
}

func (h *handler) campaignInviteActorFromParticipant(participant *statev1.Participant) *campaignInviteActor {
	if participant == nil {
		return nil
	}
	participantID := strings.TrimSpace(participant.GetId())
	if participantID == "" {
		return nil
	}
	return &campaignInviteActor{
		participantID:    participantID,
		canManageInvites: canManageCampaignAccess(participant.GetCampaignAccess()),
	}
}

func (h *handler) campaignParticipant(ctx context.Context, campaignID string, sess *session) (*statev1.Participant, error) {
	// campaignParticipant maps an access token to the participant record in the
	// campaign, with pagination across participant pages if needed.
	if h == nil || h.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	userID, err := h.sessionUserIDForSession(ctx, sess)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	return h.campaignParticipantByUserID(grpcauthctx.WithUserID(ctx, userID), campaignID, userID)
}

func (h *handler) campaignParticipantByUserID(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
	if h == nil || h.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, nil
	}

	pageToken := ""
	for {
		resp, err := h.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	return nil, nil
}
