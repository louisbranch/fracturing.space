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

func (g GRPCGateway) CreateInvite(ctx context.Context, campaignID string, input campaignapp.CreateInviteInput) error {
	if g.InviteClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}

	_, err := g.InviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: strings.TrimSpace(input.RecipientUserID),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_create_invite",
			FallbackMessage: "failed to create invite",
		})
	}
	return nil
}

func (g GRPCGateway) RevokeInvite(ctx context.Context, campaignID string, input campaignapp.RevokeInviteInput) error {
	if g.InviteClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	inviteID := strings.TrimSpace(input.InviteID)
	if inviteID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.invite_id_is_required", "invite id is required")
	}

	_, err := g.InviteClient.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: inviteID})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_revoke_invite",
			FallbackMessage: "failed to revoke invite",
		})
	}
	return nil
}
