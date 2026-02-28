package campaigns

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
)

func (g grpcGateway) CampaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	if g.inviteClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []CampaignInvite{}, nil
	}

	return grpcpaging.CollectPages[CampaignInvite, *statev1.Invite](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Invite, string, error) {
			resp, err := g.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
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
			return resp.GetInvites(), resp.GetNextPageToken(), nil
		},
		func(invite *statev1.Invite) (CampaignInvite, bool) {
			if invite == nil {
				return CampaignInvite{}, false
			}
			return CampaignInvite{
				ID:              strings.TrimSpace(invite.GetId()),
				ParticipantID:   strings.TrimSpace(invite.GetParticipantId()),
				RecipientUserID: strings.TrimSpace(invite.GetRecipientUserId()),
				Status:          inviteStatusLabel(invite.GetStatus()),
			}, true
		},
	)
}

// TODO(mutation-activation): see gateway_grpc_sessions.go for activation criteria.
func (g grpcGateway) CreateInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite creation is not implemented")
}

func (g grpcGateway) RevokeInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite revocation is not implemented")
}
