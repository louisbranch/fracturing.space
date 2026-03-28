package aitransport

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deps holds the explicit dependencies for the AI transport subpackage.
type Deps struct {
	Campaign           storage.CampaignStore
	Session            storage.SessionStore
	Participant        storage.ParticipantStore
	SessionInteraction storage.SessionInteractionStore
	SessionGrantConfig aisessiongrant.Config
}

type application struct {
	stores             applicationStores
	idGenerator        func() (string, error)
	sessionGrantConfig aisessiongrant.Config
}

type applicationStores struct {
	Campaign           storage.CampaignStore
	Session            storage.SessionStore
	Participant        storage.ParticipantStore
	SessionInteraction storage.SessionInteractionStore
}

func newApplicationWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) application {
	if clock == nil {
		clock = time.Now
	}
	sessionGrantConfig := deps.SessionGrantConfig
	sessionGrantConfig.Now = clock
	return application{
		stores: applicationStores{
			Campaign:           deps.Campaign,
			Session:            deps.Session,
			Participant:        deps.Participant,
			SessionInteraction: deps.SessionInteraction,
		},
		idGenerator:        idGenerator,
		sessionGrantConfig: sessionGrantConfig,
	}
}

func (a application) IssueCampaignAISessionGrant(
	ctx context.Context,
	campaignID string,
	sessionID string,
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

	activeSession, err := a.stores.Session.GetActiveSession(ctx, campaignID)
	if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get active session"); lookupErr != nil {
		return nil, lookupErr
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign session is not active")
	}
	if strings.TrimSpace(activeSession.ID) != sessionID {
		return nil, status.Error(codes.FailedPrecondition, "requested session does not match active campaign session")
	}
	participantID, err := a.campaignAIParticipantID(ctx, campaignID, sessionID)
	if err != nil {
		return nil, err
	}
	if participantID == "" {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai gm participant is unavailable")
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
		ParticipantID:   participantID,
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
			AuthEpoch:       claims.AuthEpoch,
			IssuedAt:        timestamppb.New(claims.IssuedAt),
			ExpiresAt:       timestamppb.New(claims.ExpiresAt),
			IssuedForUserId: claims.IssuedForUserID,
			ParticipantId:   claims.ParticipantID,
		},
	}, nil
}

func (a application) GetCampaignAIBindingUsage(
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

func (a application) GetCampaignAIAuthState(
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
	} else if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get active session"); lookupErr != nil {
		return nil, lookupErr
	}
	participantID := ""
	if activeSessionID != "" {
		participantID, err = a.campaignAIParticipantID(ctx, campaignID, activeSessionID)
		if err != nil {
			return nil, err
		}
	}

	return &campaignv1.GetCampaignAIAuthStateResponse{
		CampaignId:      campaignID,
		AiAgentId:       strings.TrimSpace(campaignRecord.AIAgentID),
		ActiveSessionId: activeSessionID,
		AuthEpoch:       campaignRecord.AIAuthEpoch,
		ParticipantId:   participantID,
	}, nil
}

func (a application) campaignAIParticipantID(ctx context.Context, campaignID, sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", nil
	}
	if a.stores.SessionInteraction == nil {
		return "", status.Error(codes.Internal, "session interaction store is not configured")
	}
	if a.stores.Participant == nil {
		return "", status.Error(codes.Internal, "participant store is not configured")
	}

	interaction, err := a.stores.SessionInteraction.GetSessionInteraction(ctx, campaignID, sessionID)
	if err != nil {
		if grpcerror.OptionalLookupErrorContext(ctx, err, "get session interaction") == nil {
			return "", nil
		}
		return "", grpcerror.OptionalLookupErrorContext(ctx, err, "get session interaction")
	}

	pid := strings.TrimSpace(interaction.AITurn.OwnerParticipantID)
	if pid == "" {
		pid = strings.TrimSpace(interaction.GMAuthorityParticipantID)
	}
	if pid == "" {
		return "", nil
	}

	record, err := a.stores.Participant.GetParticipant(ctx, campaignID, pid)
	if err != nil {
		if grpcerror.OptionalLookupErrorContext(ctx, err, "get ai participant") == nil {
			return "", nil
		}
		return "", grpcerror.OptionalLookupErrorContext(ctx, err, "get ai participant")
	}
	if record.Role != participant.RoleGM || record.Controller != participant.ControllerAI {
		return "", nil
	}
	return pid, nil
}
