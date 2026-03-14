package gateway

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CampaignInvites centralizes this web behavior in one helper seam.
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

// CreateInvite executes package-scoped creation behavior for this flow.
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
	recipientUserID, err := g.resolveInviteRecipientUserID(ctx, input.RecipientUsername)
	if err != nil {
		return err
	}

	_, err = g.InviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
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

// resolveInviteRecipientUserID canonicalizes optional invite usernames into auth user IDs.
func (g GRPCGateway) resolveInviteRecipientUserID(ctx context.Context, username string) (string, error) {
	username = strings.TrimSpace(username)
	username = strings.TrimPrefix(username, "@")
	username = strings.TrimSpace(username)
	if username == "" {
		return "", nil
	}
	if g.AuthClient == nil {
		return "", apperrors.EK(apperrors.KindUnavailable, "error.web.message.auth_service_is_not_configured", "auth service client is not configured")
	}

	resp, err := g.AuthClient.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: username})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			switch statusErr.Code() {
			case codes.InvalidArgument:
				return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_is_invalid", "recipient username is invalid")
			case codes.NotFound:
				return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_was_not_found", "recipient username was not found")
			}
		}
		return "", apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_create_invite",
			FallbackMessage: "failed to create invite",
		})
	}
	if resp == nil || resp.GetUser() == nil || strings.TrimSpace(resp.GetUser().GetId()) == "" {
		return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_was_not_found", "recipient username was not found")
	}
	return strings.TrimSpace(resp.GetUser().GetId()), nil
}

// RevokeInvite applies this package workflow transition.
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
