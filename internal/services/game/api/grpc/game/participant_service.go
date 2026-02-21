package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListParticipantsPageSize = 10
	maxListParticipantsPageSize     = 10
)

// ParticipantService implements the game.v1.ParticipantService gRPC API.
type ParticipantService struct {
	campaignv1.UnimplementedParticipantServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewParticipantService creates a ParticipantService with default dependencies.
func NewParticipantService(stores Stores) *ParticipantService {
	return &ParticipantService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateParticipant creates a participant (GM or player) for a campaign.
func (s *ParticipantService) CreateParticipant(ctx context.Context, in *campaignv1.CreateParticipantRequest) (*campaignv1.CreateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create participant request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	created, err := newParticipantApplication(s).CreateParticipant(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateParticipantResponse{Participant: participantToProto(created)}, nil
}

// UpdateParticipant updates a participant.
func (s *ParticipantService) UpdateParticipant(ctx context.Context, in *campaignv1.UpdateParticipantRequest) (*campaignv1.UpdateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update participant request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newParticipantApplication(s).UpdateParticipant(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.UpdateParticipantResponse{Participant: participantToProto(updated)}, nil
}

// DeleteParticipant deletes a participant.
func (s *ParticipantService) DeleteParticipant(ctx context.Context, in *campaignv1.DeleteParticipantRequest) (*campaignv1.DeleteParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete participant request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	current, err := newParticipantApplication(s).DeleteParticipant(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.DeleteParticipantResponse{Participant: participantToProto(current)}, nil
}

// ListParticipants returns a page of participant records for a campaign.
func (s *ParticipantService) ListParticipants(ctx context.Context, in *campaignv1.ListParticipantsRequest) (*campaignv1.ListParticipantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list participants request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
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
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, c); err != nil {
		return nil, err
	}

	p, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &campaignv1.GetParticipantResponse{
		Participant: participantToProto(p),
	}, nil
}
