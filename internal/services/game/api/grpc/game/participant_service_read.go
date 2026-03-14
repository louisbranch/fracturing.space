package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListParticipants returns a page of participant records for a campaign.
func (s *ParticipantService) ListParticipants(ctx context.Context, in *campaignv1.ListParticipantsRequest) (*campaignv1.ListParticipantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list participants request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListParticipantsPageSize,
		Max:     maxListParticipantsPageSize,
	})

	page, err := s.stores.Participant.ListParticipants(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list participants", err)
	}

	response := &campaignv1.ListParticipantsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Participants) == 0 {
		return response, nil
	}

	response.Participants = make([]*campaignv1.Participant, 0, len(page.Participants))
	for _, p := range page.Participants {
		response.Participants = append(response.Participants, participantToProto(p))
	}

	return response, nil
}

// GetParticipant returns a participant by campaign ID and participant ID.
func (s *ParticipantService) GetParticipant(ctx context.Context, in *campaignv1.GetParticipantRequest) (*campaignv1.GetParticipantResponse, error) {
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

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	p, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.GetParticipantResponse{
		Participant: participantToProto(p),
	}, nil
}
