package game

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CampaignAIService implements internal Game<=>AI/Game<=>Chat contracts.
type CampaignAIService struct {
	campaignv1.UnimplementedCampaignAIServiceServer
	stores             Stores
	clock              func() time.Time
	idGenerator        func() (string, error)
	sessionGrantConfig aisessiongrant.Config
}

// NewCampaignAIService creates a CampaignAIService with configured grant signing.
func NewCampaignAIService(stores Stores, sessionGrantConfig aisessiongrant.Config) *CampaignAIService {
	return &CampaignAIService{
		stores:             stores,
		clock:              time.Now,
		idGenerator:        id.NewID,
		sessionGrantConfig: sessionGrantConfig,
	}
}

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

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	aiAgentID := strings.TrimSpace(in.GetAiAgentId())
	if aiAgentID == "" {
		return nil, status.Error(codes.InvalidArgument, "ai agent id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
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
		return nil, status.Errorf(codes.Internal, "get active session: %v", err)
	}
	if strings.TrimSpace(activeSession.ID) != sessionID {
		return nil, status.Error(codes.FailedPrecondition, "requested session does not match active campaign session")
	}

	grantID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate grant id: %v", err)
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
		return nil, status.Errorf(codes.Internal, "issue ai session grant: %v", err)
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

// GetCampaignAIBindingUsage returns campaign usage for one bound AI agent.
func (s *CampaignAIService) GetCampaignAIBindingUsage(ctx context.Context, in *campaignv1.GetCampaignAIBindingUsageRequest) (*campaignv1.GetCampaignAIBindingUsageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign ai binding usage request is required")
	}
	aiAgentID := strings.TrimSpace(in.GetAiAgentId())
	if aiAgentID == "" {
		return nil, status.Error(codes.InvalidArgument, "ai agent id is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	bindingReader, ok := s.stores.Campaign.(storage.CampaignAIBindingReader)
	if !ok {
		return nil, status.Error(codes.Internal, "campaign ai binding reader is not configured")
	}

	campaignIDs, err := bindingReader.ListCampaignIDsByAIAgent(ctx, aiAgentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaign ids by ai agent: %v", err)
	}
	return &campaignv1.GetCampaignAIBindingUsageResponse{
		ActiveCampaignCount: int32(len(campaignIDs)),
		CampaignIds:         campaignIDs,
	}, nil
}

// GetCampaignAIAuthState returns current campaign AI authorization state.
func (s *CampaignAIService) GetCampaignAIAuthState(ctx context.Context, in *campaignv1.GetCampaignAIAuthStateRequest) (*campaignv1.GetCampaignAIAuthStateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign ai auth state request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
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
