package aitransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IssueCampaignAISessionGrant issues a signed session-scoped AI orchestration grant.
func (s *Service) IssueCampaignAISessionGrant(ctx context.Context, in *campaignv1.IssueCampaignAISessionGrantRequest) (*campaignv1.IssueCampaignAISessionGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "issue campaign ai session grant request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	return s.app.IssueCampaignAISessionGrant(ctx, campaignID, sessionID)
}
