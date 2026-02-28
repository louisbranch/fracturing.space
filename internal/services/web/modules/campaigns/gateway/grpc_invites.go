package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
)

func (g GRPCGateway) CampaignInvites(ctx context.Context, campaignID string) ([]campaignapp.CampaignInvite, error) {
	if g.InviteClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignInvite{}, nil
	}

	return grpcpaging.CollectPages[campaignapp.CampaignInvite, *statev1.Invite](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Invite, string, error) {
			resp, err := g.InviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
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
		func(invite *statev1.Invite) (campaignapp.CampaignInvite, bool) {
			if invite == nil {
				return campaignapp.CampaignInvite{}, false
			}
			return campaignapp.CampaignInvite{
				ID:              strings.TrimSpace(invite.GetId()),
				ParticipantID:   strings.TrimSpace(invite.GetParticipantId()),
				RecipientUserID: strings.TrimSpace(invite.GetRecipientUserId()),
				Status:          inviteStatusLabel(invite.GetStatus()),
			}, true
		},
	)
}

// TODO(mutation-activation): see gateway_grpc_sessions.go for activation criteria.
func (g GRPCGateway) CreateInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite creation is not implemented")
}

func (g GRPCGateway) RevokeInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaign invite revocation is not implemented")
}
