package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateParticipant creates a participant (GM or player) for a campaign.
func (s *ParticipantService) CreateParticipant(ctx context.Context, in *campaignv1.CreateParticipantRequest) (*campaignv1.CreateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	created, err := newParticipantApplication(s).CreateParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.CreateParticipantResponse{Participant: participantToProto(created)}, nil
}

// UpdateParticipant updates a participant.
func (s *ParticipantService) UpdateParticipant(ctx context.Context, in *campaignv1.UpdateParticipantRequest) (*campaignv1.UpdateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := newParticipantApplication(s).UpdateParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.UpdateParticipantResponse{Participant: participantToProto(updated)}, nil
}

// DeleteParticipant deletes a participant.
func (s *ParticipantService) DeleteParticipant(ctx context.Context, in *campaignv1.DeleteParticipantRequest) (*campaignv1.DeleteParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	current, err := newParticipantApplication(s).DeleteParticipant(ctx, campaignID, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.DeleteParticipantResponse{Participant: participantToProto(current)}, nil
}
