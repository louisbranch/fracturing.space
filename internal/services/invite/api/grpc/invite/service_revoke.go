package invite

import (
	"context"
	"errors"
	"strings"

	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) RevokeInvite(ctx context.Context, in *invitev1.RevokeInviteRequest) (*invitev1.RevokeInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite_id is required")
	}

	inv, err := s.store.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "invite not found")
		}
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}
	if inv.Status != storage.StatusPending {
		return nil, status.Errorf(codes.FailedPrecondition, "invite already %s", inv.Status)
	}

	now := s.clock()
	if err := s.store.UpdateInviteStatus(ctx, inviteID, storage.StatusRevoked, now); err != nil {
		return nil, status.Errorf(codes.Internal, "update invite status: %v", err)
	}

	updatedInvite := inv
	updatedInvite.Status = storage.StatusRevoked
	updatedInvite.UpdatedAt = now

	return &invitev1.RevokeInviteResponse{Invite: inviteToProto(updatedInvite)}, nil
}
