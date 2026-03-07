package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateInvite creates a seat-targeted invite.
func (s *InviteService) CreateInvite(ctx context.Context, in *campaignv1.CreateInviteRequest) (*campaignv1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create invite request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	inv, err := newInviteApplication(s).CreateInvite(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ClaimInvite claims a seat-targeted invite.
func (s *InviteService) ClaimInvite(ctx context.Context, in *campaignv1.ClaimInviteRequest) (*campaignv1.ClaimInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "claim invite request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	inv, participantRecord, err := newInviteApplication(s).ClaimInvite(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
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
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}

	updated, err := newInviteApplication(s).RevokeInvite(ctx, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.RevokeInviteResponse{Invite: inviteToProto(updated)}, nil
}
