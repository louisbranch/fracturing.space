package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCampaignAIAuthState returns current campaign AI authorization state.
func (s *CampaignAIService) GetCampaignAIAuthState(ctx context.Context, in *campaignv1.GetCampaignAIAuthStateRequest) (*campaignv1.GetCampaignAIAuthStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign ai auth state request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	activeSessionID := ""
	activeSession, err := s.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		activeSessionID = strings.TrimSpace(activeSession.ID)
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get active session: %v", err)
	}

	return &campaignv1.GetCampaignAIAuthStateResponse{
		CampaignId:      campaignID,
		AiAgentId:       strings.TrimSpace(campaignRecord.AIAgentID),
		ActiveSessionId: activeSessionID,
		AuthEpoch:       campaignRecord.AIAuthEpoch,
	}, nil
}
