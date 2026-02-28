package campaigns

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
)

func (g grpcGateway) CampaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	if g.participantClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignParticipant{}, nil
	}

	return grpcpaging.CollectPages[CampaignParticipant, *statev1.Participant](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Participant, string, error) {
			resp, err := g.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetParticipants(), resp.GetNextPageToken(), nil
		},
		func(participant *statev1.Participant) (CampaignParticipant, bool) {
			if participant == nil {
				return CampaignParticipant{}, false
			}
			participantID := strings.TrimSpace(participant.GetId())
			avatarEntityID := participantID
			if avatarEntityID == "" {
				avatarEntityID = strings.TrimSpace(participant.GetUserId())
			}
			if avatarEntityID == "" {
				avatarEntityID = campaignID
			}
			return CampaignParticipant{
				ID:             participantID,
				UserID:         strings.TrimSpace(participant.GetUserId()),
				Name:           participantDisplayName(participant),
				Role:           participantRoleLabel(participant.GetRole()),
				CampaignAccess: participantCampaignAccessLabel(participant.GetCampaignAccess()),
				Controller:     participantControllerLabel(participant.GetController()),
				Pronouns:       pronouns.FromProto(participant.GetPronouns()),
				AvatarURL: websupport.AvatarImageURL(
					g.assetBaseURL,
					catalog.AvatarRoleParticipant,
					avatarEntityID,
					strings.TrimSpace(participant.GetAvatarSetId()),
					strings.TrimSpace(participant.GetAvatarAssetId()),
				),
			}, true
		},
	)
}
