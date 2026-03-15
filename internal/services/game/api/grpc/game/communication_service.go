package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommunicationService exposes game-owned communication context to transport
// layers so chat/web do not infer gameplay routing rules on their own.
type CommunicationService struct {
	campaignv1.UnimplementedCommunicationServiceServer
	app communicationApplication
}

// NewCommunicationService creates a CommunicationService with projection-backed
// read dependencies.
func NewCommunicationService(stores Stores) *CommunicationService {
	return newCommunicationServiceWithDependencies(stores, id.NewID)
}

func newCommunicationServiceWithDependencies(
	stores Stores,
	idGenerator func() (string, error),
) *CommunicationService {
	return &CommunicationService{
		app: newCommunicationApplicationWithDependencies(stores, idGenerator),
	}
}

// GetCommunicationContext returns caller-specific communication metadata for a campaign.
func (s *CommunicationService) GetCommunicationContext(ctx context.Context, in *campaignv1.GetCommunicationContextRequest) (*campaignv1.GetCommunicationContextResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "communication context request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	contextState, err := s.app.GetCommunicationContext(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return &campaignv1.GetCommunicationContextResponse{Context: contextState}, nil
}
