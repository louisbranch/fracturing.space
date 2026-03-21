package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RunCampaignTurn validates a game-issued session grant and executes one GM turn.
func (s *Service) RunCampaignTurn(ctx context.Context, in *aiv1.RunCampaignTurnRequest) (*aiv1.RunCampaignTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "run campaign turn request is required")
	}
	if s.campaignTurnRunner == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign turn runner is unavailable")
	}
	if s.sessionGrantConfig == nil {
		return nil, status.Error(codes.FailedPrecondition, "ai session grant validation is unavailable")
	}
	if s.gameCampaignAIClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai auth state client is unavailable")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	grant := strings.TrimSpace(in.GetSessionGrant())
	if grant == "" {
		return nil, status.Error(codes.InvalidArgument, "session_grant is required")
	}

	claims, err := aisessiongrant.Validate(*s.sessionGrantConfig, grant)
	if err != nil {
		switch {
		case errors.Is(err, aisessiongrant.ErrExpired):
			return nil, status.Error(codes.PermissionDenied, "session grant is expired")
		case errors.Is(err, aisessiongrant.ErrInvalid):
			return nil, status.Error(codes.PermissionDenied, "session grant is invalid")
		default:
			return nil, status.Errorf(codes.Internal, "validate session grant: %v", err)
		}
	}

	state, err := s.gameCampaignAIClient.GetCampaignAIAuthState(ctx, &gamev1.GetCampaignAIAuthStateRequest{
		CampaignId: claims.CampaignID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get campaign ai auth state: %v", err)
	}
	if staleGrant(claims, state) {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai session grant is stale")
	}

	agentID := strings.TrimSpace(state.GetAiAgentId())
	if agentID == "" {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai runtime is unavailable")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "campaign ai runtime is unavailable")
		}
		return nil, status.Errorf(codes.Internal, "get campaign ai runtime: %v", err)
	}
	if !agent.ParseStatus(agentRecord.Status).IsActive() {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai runtime is inactive")
	}
	if s.campaignArtifactManager != nil {
		if _, err := s.campaignArtifactManager.EnsureDefaultArtifacts(ctx, claims.CampaignID, ""); err != nil {
			return nil, status.Errorf(codes.Internal, "ensure campaign artifacts: %v", err)
		}
	}

	provider := providerFromString(agentRecord.Provider)
	adapter, ok := s.providerToolAdapters[provider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign ai provider adapter is unavailable")
	}

	token, err := s.resolveAgentInvokeToken(ctx, strings.TrimSpace(agentRecord.OwnerUserID), agentRecord)
	if err != nil {
		return nil, err
	}

	result, err := s.campaignTurnRunner.Run(ctx, orchestration.Input{
		CampaignID:       claims.CampaignID,
		SessionID:        claims.SessionID,
		ParticipantID:    strings.TrimSpace(state.GetParticipantId()),
		Input:            strings.TrimSpace(in.GetInput()),
		Model:            strings.TrimSpace(agentRecord.Model),
		ReasoningEffort:  strings.TrimSpace(in.GetReasoningEffort()),
		Instructions:     strings.TrimSpace(agentRecord.Instructions),
		CredentialSecret: token,
		Provider:         adapter,
	})
	if err != nil {
		return nil, campaignTurnGRPCError(err)
	}
	if strings.TrimSpace(result.OutputText) == "" {
		return nil, campaignTurnGRPCError(orchestration.ErrEmptyOutput)
	}
	return &aiv1.RunCampaignTurnResponse{
		OutputText: result.OutputText,
		Provider:   providerToProto(agentRecord.Provider),
		Model:      agentRecord.Model,
		Usage:      usageToProto(result.Usage),
	}, nil
}

func campaignTurnGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return apperrors.HandleError(err, apperrors.DefaultLocale)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.HandleError(
			apperrors.Wrap(apperrors.CodeAIOrchestrationTimedOut, "campaign turn timed out", err),
			apperrors.DefaultLocale,
		)
	}
	if errors.Is(err, context.Canceled) {
		return apperrors.HandleError(
			apperrors.Wrap(apperrors.CodeAIOrchestrationCanceled, "campaign turn canceled", err),
			apperrors.DefaultLocale,
		)
	}
	return status.Errorf(codes.Internal, "run campaign turn: %v", err)
}

func staleGrant(claims aisessiongrant.Claims, state *gamev1.GetCampaignAIAuthStateResponse) bool {
	if state == nil {
		return true
	}
	if strings.TrimSpace(state.GetCampaignId()) != strings.TrimSpace(claims.CampaignID) {
		return true
	}
	if strings.TrimSpace(state.GetActiveSessionId()) != strings.TrimSpace(claims.SessionID) {
		return true
	}
	if strings.TrimSpace(state.GetParticipantId()) != strings.TrimSpace(claims.ParticipantID) {
		return true
	}
	return state.GetAuthEpoch() != claims.AuthEpoch
}
