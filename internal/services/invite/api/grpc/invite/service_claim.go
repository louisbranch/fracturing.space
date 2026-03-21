package invite

import (
	"context"
	"errors"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *Service) ClaimInvite(ctx context.Context, in *invitev1.ClaimInviteRequest) (*invitev1.ClaimInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite_id is required")
	}
	joinGrantToken := strings.TrimSpace(in.GetJoinGrant())
	if joinGrantToken == "" {
		return nil, status.Error(codes.InvalidArgument, "join_grant is required")
	}

	// Extract user ID from gRPC metadata (set by the web service caller).
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user identity is required")
	}

	inv, err := s.store.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "invite not found")
		}
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}
	if inv.CampaignID != campaignID {
		return nil, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if inv.Status != storage.StatusPending {
		return nil, status.Errorf(codes.FailedPrecondition, "invite already %s", inv.Status)
	}

	// Check recipient restriction.
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return nil, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}

	// Verify the join grant JWT.
	if s.verifier != nil {
		_, err = s.verifier.Validate(joinGrantToken, joingrant.Expectation{
			CampaignID: campaignID,
			InviteID:   inviteID,
			UserID:     userID,
		})
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "invalid join grant: %v", err)
		}
	}

	// Call game service to bind the participant seat. Propagate the original
	// gRPC status (e.g. AlreadyExists for duplicate claim) so callers can map
	// domain errors to user-facing messages.
	bindResp, err := s.game.BindParticipant(ctx, &gamev1.BindParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: inv.ParticipantID,
		UserId:        userID,
	})
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "bind participant: %v", err)
	}

	// Mark invite as claimed.
	now := s.clock()
	if err := s.store.UpdateInviteStatus(ctx, inviteID, storage.StatusClaimed, now); err != nil {
		return nil, status.Errorf(codes.Internal, "update invite status: %v", err)
	}

	if s.outbox != nil {
		inv.Status = storage.StatusClaimed
		enqueueInviteEvent(ctx, s.outbox, s.idGenerator, now, outboxEventClaimed, inv)
	}

	updatedInvite := inv
	updatedInvite.Status = storage.StatusClaimed
	updatedInvite.UpdatedAt = now

	var participantSummary *invitev1.InviteParticipantSummary
	if bindResp.GetParticipant() != nil {
		p := bindResp.GetParticipant()
		participantSummary = &invitev1.InviteParticipantSummary{
			Id:     p.GetId(),
			Name:   p.GetName(),
			Role:   p.GetRole().String(),
			UserId: p.GetUserId(),
		}
	}

	return &invitev1.ClaimInviteResponse{
		Invite:      inviteToProto(updatedInvite),
		Participant: participantSummary,
	}, nil
}

// userIDFromContext extracts the user ID from incoming gRPC metadata.
func userIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(grpcmeta.FirstMetadataValue(md, grpcmeta.UserIDHeader))
}
