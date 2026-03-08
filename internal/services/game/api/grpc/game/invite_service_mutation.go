package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateInvite creates a seat-targeted invite.
func (s *InviteService) CreateInvite(ctx context.Context, in *campaignv1.CreateInviteRequest) (*campaignv1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create invite request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	inv, err := newInviteApplication(s).CreateInvite(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ClaimInvite claims a seat-targeted invite.
func (s *InviteService) ClaimInvite(ctx context.Context, in *campaignv1.ClaimInviteRequest) (*campaignv1.ClaimInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "claim invite request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	inv, participantRecord, err := newInviteApplication(s).ClaimInvite(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ClaimInviteResponse{
		Invite:      inviteToProto(inv),
		Participant: participantToProto(participantRecord),
	}, nil
}

// RevokeInvite revokes an invite.
func (s *InviteService) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (*campaignv1.RevokeInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke invite request is required")
	}
	_, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return nil, err
	}

	updated, err := newInviteApplication(s).RevokeInvite(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.RevokeInviteResponse{Invite: inviteToProto(updated)}, nil
}
