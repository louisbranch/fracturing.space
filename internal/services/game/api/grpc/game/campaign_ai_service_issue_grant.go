package game

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IssueCampaignAISessionGrant issues a signed session-scoped AI turn relay grant.
func (s *CampaignAIService) IssueCampaignAISessionGrant(ctx context.Context, in *campaignv1.IssueCampaignAISessionGrantRequest) (*campaignv1.IssueCampaignAISessionGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "issue campaign ai session grant request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	aiAgentID, err := validate.RequiredID(in.GetAiAgentId(), "ai agent id")
	if err != nil {
		return nil, err
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if campaignRecord.GmMode != campaign.GmModeAI && campaignRecord.GmMode != campaign.GmModeHybrid {
		return nil, status.Error(codes.FailedPrecondition, "campaign gm mode does not support ai relay")
	}
	boundAgentID := strings.TrimSpace(campaignRecord.AIAgentID)
	if boundAgentID == "" {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai binding is required")
	}
	if boundAgentID != aiAgentID {
		return nil, status.Error(codes.FailedPrecondition, "requested ai agent does not match campaign binding")
	}

	activeSession, err := s.stores.Session.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "campaign session is not active")
		}
		return nil, grpcerror.Internal("get active session", err)
	}
	if strings.TrimSpace(activeSession.ID) != sessionID {
		return nil, status.Error(codes.FailedPrecondition, "requested session does not match active campaign session")
	}

	grantID, err := s.idGenerator()
	if err != nil {
		return nil, grpcerror.Internal("generate grant id", err)
	}
	issuedForUserID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	grantToken, claims, err := aisessiongrant.Issue(s.sessionGrantConfig, aisessiongrant.IssueInput{
		GrantID:         grantID,
		CampaignID:      campaignID,
		SessionID:       sessionID,
		AIAgentID:       aiAgentID,
		AuthEpoch:       campaignRecord.AIAuthEpoch,
		IssuedForUserID: issuedForUserID,
	})
	if err != nil {
		return nil, grpcerror.Internal("issue ai session grant", err)
	}

	return &campaignv1.IssueCampaignAISessionGrantResponse{
		Grant: &campaignv1.AISessionGrant{
			Token:           grantToken,
			GrantId:         claims.GrantID,
			CampaignId:      claims.CampaignID,
			SessionId:       claims.SessionID,
			AiAgentId:       claims.AIAgentID,
			AuthEpoch:       claims.AuthEpoch,
			IssuedAt:        timestamppb.New(claims.IssuedAt),
			ExpiresAt:       timestamppb.New(claims.ExpiresAt),
			IssuedForUserId: claims.IssuedForUserID,
		},
	}, nil
}
