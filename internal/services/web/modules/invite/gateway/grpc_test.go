package gateway

import (
	"context"
	"errors"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCGatewayRequiresBothClients(t *testing.T) {
	t.Parallel()

	if got := NewGRPCGateway(nil, &authClientStub{}); got != nil {
		t.Fatal("expected nil gateway when invite client is missing")
	}
	if got := NewGRPCGateway(&inviteClientStub{}, nil); got != nil {
		t.Fatal("expected nil gateway when auth client is missing")
	}
	if got := NewGRPCGateway(&inviteClientStub{}, &authClientStub{}); got == nil {
		t.Fatal("expected non-nil gateway when both clients are configured")
	}
}

func TestGetPublicInviteMapsResponse(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Invites: &inviteClientStub{
			publicResp: &gamev1.GetPublicInviteResponse{
				Invite: &gamev1.Invite{
					Id:              "inv-1",
					CampaignId:      "camp-1",
					ParticipantId:   "part-1",
					RecipientUserId: "user-1",
					Status:          gamev1.InviteStatus_DECLINED,
				},
				Campaign:      &gamev1.PublicInviteCampaign{Name: "Skyfall", Status: gamev1.CampaignStatus_ACTIVE},
				Participant:   &gamev1.Participant{Name: "Scout"},
				CreatedByUser: &authv1.User{Id: "creator-1", Username: "gm"},
			},
		},
		Auth: &authClientStub{},
	}

	invite, err := gateway.GetPublicInvite(context.Background(), " inv-1 ")
	if err != nil {
		t.Fatalf("GetPublicInvite() error = %v", err)
	}
	if invite.InviteID != "inv-1" || invite.CampaignName != "Skyfall" || invite.ParticipantName != "Scout" || invite.InviterUsername != "gm" || invite.Status != inviteapp.InviteStatusDeclined {
		t.Fatalf("invite = %+v, want mapped fields", invite)
	}
}

func TestGetPublicInviteHandlesMissingInvite(t *testing.T) {
	t.Parallel()

	_, err := GRPCGateway{Invites: &inviteClientStub{}, Auth: &authClientStub{}}.GetPublicInvite(context.Background(), "inv-1")
	if got := apperrors.HTTPStatus(err); got != 404 {
		t.Fatalf("HTTPStatus(err) = %d, want 404", got)
	}
}

func TestAcceptInviteIssuesGrantAndClaimsWithUserContext(t *testing.T) {
	t.Parallel()

	invites := &inviteClientStub{}
	auth := &authClientStub{}
	gateway := GRPCGateway{Invites: invites, Auth: auth}

	err := gateway.AcceptInvite(context.Background(), "user-1", inviteapp.PublicInvite{
		InviteID:      "inv-1",
		CampaignID:    "camp-1",
		ParticipantID: "part-1",
	})
	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}
	if auth.lastIssue == nil || auth.lastIssue.GetUserId() != "user-1" || auth.lastIssue.GetInviteId() != "inv-1" {
		t.Fatalf("IssueJoinGrant request = %+v, want mapped user and invite ids", auth.lastIssue)
	}
	if invites.lastClaim == nil || invites.lastClaim.GetJoinGrant() != "grant" {
		t.Fatalf("ClaimInvite request = %+v, want join grant", invites.lastClaim)
	}
	if got := grpcauthctx.UserIDFromOutgoingContext(invites.claimCtx); got != "user-1" {
		t.Fatalf("claim user metadata = %q, want %q", got, "user-1")
	}
}

func TestAcceptInviteMapsGrantError(t *testing.T) {
	t.Parallel()

	err := GRPCGateway{
		Invites: &inviteClientStub{},
		Auth:    &authClientStub{issueErr: errors.New("boom")},
	}.AcceptInvite(context.Background(), "user-1", inviteapp.PublicInvite{InviteID: "inv-1", CampaignID: "camp-1", ParticipantID: "part-1"})
	if got := apperrors.HTTPStatus(err); got != 503 {
		t.Fatalf("HTTPStatus(err) = %d, want 503", got)
	}
}

func TestAcceptInviteMapsExistingParticipantConflict(t *testing.T) {
	t.Parallel()

	conflictStatus, err := status.New(codes.AlreadyExists, "participant user already claimed").WithDetails(&errdetails.ErrorInfo{
		Reason: "PARTICIPANT_USER_ALREADY_CLAIMED",
	})
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	err = GRPCGateway{
		Invites: &inviteClientStub{claimErr: conflictStatus.Err()},
		Auth:    &authClientStub{},
	}.AcceptInvite(context.Background(), "user-1", inviteapp.PublicInvite{InviteID: "inv-1", CampaignID: "camp-1", ParticipantID: "part-1"})
	if got := apperrors.HTTPStatus(err); got != 409 {
		t.Fatalf("HTTPStatus(err) = %d, want 409", got)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.invite_claim_user_already_in_campaign" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.invite_claim_user_already_in_campaign")
	}
}

func TestAcceptInviteMapsClaimedSeatConflict(t *testing.T) {
	t.Parallel()

	err := GRPCGateway{
		Invites: &inviteClientStub{claimErr: status.Error(codes.FailedPrecondition, "participant already claimed")},
		Auth:    &authClientStub{},
	}.AcceptInvite(context.Background(), "user-1", inviteapp.PublicInvite{InviteID: "inv-1", CampaignID: "camp-1", ParticipantID: "part-1"})
	if got := apperrors.HTTPStatus(err); got != 409 {
		t.Fatalf("HTTPStatus(err) = %d, want 409", got)
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.invite_claim_seat_already_claimed" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.invite_claim_seat_already_claimed")
	}
}

func TestDeclineInviteSendsUserContext(t *testing.T) {
	t.Parallel()

	invites := &inviteClientStub{}
	gateway := GRPCGateway{Invites: invites, Auth: &authClientStub{}}

	if err := gateway.DeclineInvite(context.Background(), "user-1", " inv-1 "); err != nil {
		t.Fatalf("DeclineInvite() error = %v", err)
	}
	if invites.lastDecline == nil || invites.lastDecline.GetInviteId() != "inv-1" {
		t.Fatalf("DeclineInvite request = %+v, want trimmed invite id", invites.lastDecline)
	}
	if got := grpcauthctx.UserIDFromOutgoingContext(invites.declineCtx); got != "user-1" {
		t.Fatalf("decline user metadata = %q, want %q", got, "user-1")
	}
}

type inviteClientStub struct {
	publicResp  *gamev1.GetPublicInviteResponse
	publicErr   error
	claimErr    error
	declineErr  error
	claimCtx    context.Context
	declineCtx  context.Context
	lastClaim   *gamev1.ClaimInviteRequest
	lastDecline *gamev1.DeclineInviteRequest
}

func (s inviteClientStub) GetPublicInvite(context.Context, *gamev1.GetPublicInviteRequest, ...grpc.CallOption) (*gamev1.GetPublicInviteResponse, error) {
	if s.publicErr != nil {
		return nil, s.publicErr
	}
	return s.publicResp, nil
}

func (s *inviteClientStub) ClaimInvite(ctx context.Context, req *gamev1.ClaimInviteRequest, _ ...grpc.CallOption) (*gamev1.ClaimInviteResponse, error) {
	s.claimCtx = ctx
	s.lastClaim = req
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	return &gamev1.ClaimInviteResponse{}, nil
}

func (s *inviteClientStub) DeclineInvite(ctx context.Context, req *gamev1.DeclineInviteRequest, _ ...grpc.CallOption) (*gamev1.DeclineInviteResponse, error) {
	s.declineCtx = ctx
	s.lastDecline = req
	if s.declineErr != nil {
		return nil, s.declineErr
	}
	return &gamev1.DeclineInviteResponse{}, nil
}

type authClientStub struct {
	issueErr  error
	lastIssue *authv1.IssueJoinGrantRequest
}

func (s *authClientStub) IssueJoinGrant(_ context.Context, req *authv1.IssueJoinGrantRequest, _ ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	if s.issueErr != nil {
		return nil, s.issueErr
	}
	s.lastIssue = req
	return &authv1.IssueJoinGrantResponse{JoinGrant: "grant"}, nil
}
