package gateway

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	domainerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InviteClient exposes public invite reads and claim/decline mutations.
type InviteClient interface {
	GetPublicInvite(context.Context, *invitev1.GetPublicInviteRequest, ...grpc.CallOption) (*invitev1.GetPublicInviteResponse, error)
	ClaimInvite(context.Context, *invitev1.ClaimInviteRequest, ...grpc.CallOption) (*invitev1.ClaimInviteResponse, error)
	DeclineInvite(context.Context, *invitev1.DeclineInviteRequest, ...grpc.CallOption) (*invitev1.DeclineInviteResponse, error)
}

// AuthClient exposes the join-grant issuance needed before claiming an invite.
type AuthClient interface {
	IssueJoinGrant(context.Context, *authv1.IssueJoinGrantRequest, ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error)
}

// GRPCGateway adapts gRPC invite/auth clients to the public invite gateway contract.
type GRPCGateway struct {
	Invites InviteClient
	Auth    AuthClient
}

// NewGRPCGateway builds the production public invite gateway.
func NewGRPCGateway(invites InviteClient, auth AuthClient) inviteapp.Gateway {
	if invites == nil || auth == nil {
		return nil
	}
	return GRPCGateway{Invites: invites, Auth: auth}
}

// GetPublicInvite translates the public game invite read into the web app model.
func (g GRPCGateway) GetPublicInvite(ctx context.Context, inviteID string) (inviteapp.PublicInvite, error) {
	resp, err := g.Invites.GetPublicInvite(ctx, &invitev1.GetPublicInviteRequest{InviteId: strings.TrimSpace(inviteID)})
	if err != nil {
		return inviteapp.PublicInvite{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_load_invite",
			FallbackMessage: "failed to load invite",
		})
	}
	if resp == nil || resp.GetInvite() == nil {
		return inviteapp.PublicInvite{}, apperrors.E(apperrors.KindNotFound, "invite not found")
	}
	invite := resp.GetInvite()
	return inviteapp.PublicInvite{
		InviteID:        strings.TrimSpace(invite.GetId()),
		CampaignID:      strings.TrimSpace(invite.GetCampaignId()),
		CampaignName:    strings.TrimSpace(resp.GetCampaign().GetName()),
		CampaignStatus:  strings.TrimSpace(resp.GetCampaign().GetStatus()),
		ParticipantID:   strings.TrimSpace(invite.GetParticipantId()),
		ParticipantName: strings.TrimSpace(resp.GetParticipant().GetName()),
		RecipientUserID: strings.TrimSpace(invite.GetRecipientUserId()),
		CreatedByUserID: strings.TrimSpace(resp.GetCreatedByUser().GetId()),
		InviterUsername: strings.TrimSpace(resp.GetCreatedByUser().GetUsername()),
		Status:          mapInviteStatus(invite.GetStatus()),
	}, nil
}

// AcceptInvite issues the join grant required by auth before claiming the seat
// through the game write path.
func (g GRPCGateway) AcceptInvite(ctx context.Context, viewerUserID string, invite inviteapp.PublicInvite) error {
	grantResp, err := g.Auth.IssueJoinGrant(ctx, &authv1.IssueJoinGrantRequest{
		UserId:        viewerUserID,
		CampaignId:    invite.CampaignID,
		InviteId:      invite.InviteID,
		ParticipantId: invite.ParticipantID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_accept_invite",
			FallbackMessage: "failed to accept invite",
		})
	}
	_, err = g.Invites.ClaimInvite(grpcauthctx.WithUserID(ctx, viewerUserID), &invitev1.ClaimInviteRequest{
		CampaignId: invite.CampaignID,
		InviteId:   invite.InviteID,
		JoinGrant:  grantResp.GetJoinGrant(),
	})
	if err != nil {
		if mapped := mapClaimInviteTransportError(err); mapped != nil {
			return mapped
		}
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_accept_invite",
			FallbackMessage: "failed to accept invite",
		})
	}
	return nil
}

// DeclineInvite sends the recipient-owned decline mutation through the game API.
func (g GRPCGateway) DeclineInvite(ctx context.Context, viewerUserID string, inviteID string) error {
	_, err := g.Invites.DeclineInvite(grpcauthctx.WithUserID(ctx, viewerUserID), &invitev1.DeclineInviteRequest{
		InviteId: strings.TrimSpace(inviteID),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_decline_invite",
			FallbackMessage: "failed to decline invite",
		})
	}
	return nil
}

// mapInviteStatus normalizes invite transport enums into the smaller public-web set.
func mapInviteStatus(status invitev1.InviteStatus) inviteapp.InviteStatus {
	switch status {
	case invitev1.InviteStatus_CLAIMED:
		return inviteapp.InviteStatusClaimed
	case invitev1.InviteStatus_DECLINED:
		return inviteapp.InviteStatusDeclined
	case invitev1.InviteStatus_REVOKED:
		return inviteapp.InviteStatusRevoked
	default:
		return inviteapp.InviteStatusPending
	}
}

// mapClaimInviteTransportError translates known claim conflicts into web-localized
// errors so invite accept failures render as user-facing messages instead of 500s.
func mapClaimInviteTransportError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return nil
	}

	switch grpcErrorReason(st) {
	case string(domainerrors.CodeParticipantUserAlreadyClaimed):
		return apperrors.EK(
			apperrors.KindConflict,
			"error.web.message.invite_claim_user_already_in_campaign",
			"user already has a participant in this campaign",
		)
	}

	if st.Code() != codes.FailedPrecondition {
		return nil
	}

	switch strings.TrimSpace(st.Message()) {
	case "participant already claimed", "invite already claimed":
		return apperrors.EK(
			apperrors.KindConflict,
			"error.web.message.invite_claim_seat_already_claimed",
			"invite seat has already been claimed",
		)
	default:
		return nil
	}
}

// grpcErrorReason extracts structured ErrorInfo reasons so web transport mapping
// can branch on durable domain error codes instead of ad hoc strings when present.
func grpcErrorReason(st *status.Status) string {
	if st == nil {
		return ""
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		return strings.TrimSpace(info.GetReason())
	}
	return ""
}
