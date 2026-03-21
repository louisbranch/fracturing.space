package participanttransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateParticipant creates a participant (GM or player) for a campaign.
func (s *Service) CreateParticipant(ctx context.Context, in *campaignv1.CreateParticipantRequest) (*campaignv1.CreateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	created, err := s.app.CreateParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateParticipantResponse{Participant: ParticipantToProto(created)}, nil
}

// UpdateParticipant updates a participant.
func (s *Service) UpdateParticipant(ctx context.Context, in *campaignv1.UpdateParticipantRequest) (*campaignv1.UpdateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.UpdateParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.UpdateParticipantResponse{Participant: ParticipantToProto(updated)}, nil
}

// DeleteParticipant deletes a participant.
func (s *Service) DeleteParticipant(ctx context.Context, in *campaignv1.DeleteParticipantRequest) (*campaignv1.DeleteParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	current, err := s.app.DeleteParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.DeleteParticipantResponse{Participant: ParticipantToProto(current)}, nil
}

// BindParticipant binds a user to an unoccupied participant seat.
// Internal-only: the caller is trusted to have verified authorization.
func (s *Service) BindParticipant(ctx context.Context, in *campaignv1.BindParticipantRequest) (*campaignv1.BindParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "bind participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
	if err != nil {
		return nil, err
	}
	userID, err := validate.RequiredID(in.GetUserId(), "user id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.BindParticipant(ctx, campaignID, participantID, userID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.BindParticipantResponse{Participant: ParticipantToProto(updated)}, nil
}
