package campaigntransport

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type campaignAIApplication struct {
	stores             campaignAIApplicationStores
	idGenerator        func() (string, error)
	sessionGrantConfig aisessiongrant.Config
}

type campaignAIApplicationStores struct {
	Campaign storage.CampaignStore
	Session  storage.SessionStore
}

func newCampaignAIApplicationWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) campaignAIApplication {
	if clock == nil {
		clock = time.Now
	}
	sessionGrantConfig := deps.SessionGrantConfig
	sessionGrantConfig.Now = clock
	return campaignAIApplication{
		stores: campaignAIApplicationStores{
			Campaign: deps.Campaign,
			Session:  deps.Session,
		},
		idGenerator:        idGenerator,
		sessionGrantConfig: sessionGrantConfig,
	}
}

func (a campaignAIApplication) IssueCampaignAISessionGrant(
	ctx context.Context,
	campaignID string,
	sessionID string,
	aiAgentID string,
) (*campaignv1.IssueCampaignAISessionGrantResponse, error) {
	if a.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if campaignRecord.GmMode != campaign.GmModeAI && campaignRecord.GmMode != campaign.GmModeHybrid {
		return nil, status.Error(codes.FailedPrecondition, "campaign gm mode does not support ai orchestration")
	}
	boundAgentID := strings.TrimSpace(campaignRecord.AIAgentID)
	if boundAgentID == "" {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai binding is required")
	}
	if boundAgentID != aiAgentID {
		return nil, status.Error(codes.FailedPrecondition, "requested ai agent does not match campaign binding")
	}

	activeSession, err := a.stores.Session.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "campaign session is not active")
		}
		return nil, grpcerror.Internal("get active session", err)
	}
	if strings.TrimSpace(activeSession.ID) != sessionID {
		return nil, status.Error(codes.FailedPrecondition, "requested session does not match active campaign session")
	}

	grantID, err := a.idGenerator()
	if err != nil {
		return nil, grpcerror.Internal("generate grant id", err)
	}
	issuedForUserID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	grantToken, claims, err := aisessiongrant.Issue(a.sessionGrantConfig, aisessiongrant.IssueInput{
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

func (a campaignAIApplication) GetCampaignAIBindingUsage(
	ctx context.Context,
	aiAgentID string,
) (*campaignv1.GetCampaignAIBindingUsageResponse, error) {
	if a.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	bindingReader, ok := a.stores.Campaign.(storage.CampaignAIBindingReader)
	if !ok {
		return nil, status.Error(codes.Internal, "campaign ai binding reader is not configured")
	}

	campaignIDs, err := bindingReader.ListCampaignIDsByAIAgent(ctx, aiAgentID)
	if err != nil {
		return nil, grpcerror.Internal("list campaign ids by ai agent", err)
	}
	return &campaignv1.GetCampaignAIBindingUsageResponse{
		ActiveCampaignCount: int32(len(campaignIDs)),
		CampaignIds:         campaignIDs,
	}, nil
}

func (a campaignAIApplication) GetCampaignAIAuthState(
	ctx context.Context,
	campaignID string,
) (*campaignv1.GetCampaignAIAuthStateResponse, error) {
	if a.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if a.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	activeSessionID := ""
	activeSession, err := a.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		activeSessionID = strings.TrimSpace(activeSession.ID)
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, grpcerror.Internal("get active session", err)
	}

	return &campaignv1.GetCampaignAIAuthStateResponse{
		CampaignId:      campaignID,
		AiAgentId:       strings.TrimSpace(campaignRecord.AIAgentID),
		ActiveSessionId: activeSessionID,
		AuthEpoch:       campaignRecord.AIAuthEpoch,
	}, nil
}
