package participanttransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListParticipants returns a page of participant records for a campaign.
func (s *Service) ListParticipants(ctx context.Context, in *campaignv1.ListParticipantsRequest) (*campaignv1.ListParticipantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list participants request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	page, err := s.app.ListParticipants(ctx, campaignID, in.GetPageToken(), in.GetPageSize())
	if err != nil {
		return nil, err
	}

	response := &campaignv1.ListParticipantsResponse{
		NextPageToken: page.nextPageToken,
	}
	if len(page.participants) == 0 {
		return response, nil
	}

	response.Participants = make([]*campaignv1.Participant, 0, len(page.participants))
	for _, p := range page.participants {
		response.Participants = append(response.Participants, ParticipantToProto(p))
	}

	return response, nil
}

// GetParticipant returns a participant by campaign ID and participant ID.
func (s *Service) GetParticipant(ctx context.Context, in *campaignv1.GetParticipantRequest) (*campaignv1.GetParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get participant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
	if err != nil {
		return nil, err
	}

	p, err := s.app.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetParticipantResponse{
		Participant: ParticipantToProto(p),
	}, nil
}
