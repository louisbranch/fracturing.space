package invite

import (
	"context"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CreateInvite(ctx context.Context, in *invitev1.CreateInviteRequest) (*invitev1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant_id is required")
	}
	recipientUserID := strings.TrimSpace(in.GetRecipientUserId())

	// Verify the participant seat is not already bound to a user.
	if s.game != nil {
		resp, err := s.game.GetParticipant(gameReadContext(ctx), &gamev1.GetParticipantRequest{
			CampaignId:    campaignID,
			ParticipantId: participantID,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				return nil, status.Error(codes.NotFound, "participant not found")
			}
			return nil, status.Errorf(codes.Internal, "check participant: %v", err)
		}
		if uid := strings.TrimSpace(resp.GetParticipant().GetUserId()); uid != "" {
			return nil, status.Error(codes.AlreadyExists, "participant already has a user")
		}
	}

	// If a recipient is specified, check they don't already have a seat
	// (bound participant) in this campaign.
	if recipientUserID != "" && s.game != nil {
		if userHasSeatInCampaign(gameReadContext(ctx), s.game, campaignID, recipientUserID) {
			return nil, status.Error(codes.FailedPrecondition, "recipient already has a seat in this campaign")
		}
	}

	inviteID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate invite id: %v", err)
	}

	now := s.clock()
	rec := storage.InviteRecord{
		ID:                     inviteID,
		CampaignID:             campaignID,
		ParticipantID:          participantID,
		RecipientUserID:        recipientUserID,
		Status:                 storage.StatusPending,
		CreatedByParticipantID: strings.TrimSpace(in.GetCreatedByParticipantId()),
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.store.PutInvite(ctx, rec); err != nil {
		return nil, status.Errorf(codes.Internal, "store invite: %v", err)
	}

	if s.outbox != nil {
		enqueueInviteEvent(ctx, s.outbox, s.idGenerator, now, outboxEventCreated, rec)
	}

	return &invitev1.CreateInviteResponse{Invite: inviteToProto(rec)}, nil
}
