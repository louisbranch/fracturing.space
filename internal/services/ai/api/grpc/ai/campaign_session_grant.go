package ai

import (
	"context"
	"errors"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) validateCampaignSessionGrant(
	ctx context.Context,
	sessionGrant string,
	campaignID string,
	sessionID string,
	agentID string,
) (aisessiongrant.Claims, error) {
	sessionGrant = strings.TrimSpace(sessionGrant)
	if sessionGrant == "" {
		return aisessiongrant.Claims{}, status.Error(codes.InvalidArgument, "session_grant is required")
	}
	if err := s.validateSessionGrantConfig(); err != nil {
		return aisessiongrant.Claims{}, err
	}

	claims, err := aisessiongrant.Validate(s.sessionGrantConfig, sessionGrant)
	if err != nil {
		if errors.Is(err, aisessiongrant.ErrExpired) {
			return aisessiongrant.Claims{}, status.Error(codes.PermissionDenied, "session grant is expired")
		}
		if errors.Is(err, aisessiongrant.ErrInvalid) {
			return aisessiongrant.Claims{}, status.Error(codes.PermissionDenied, "session grant is invalid")
		}
		return aisessiongrant.Claims{}, status.Errorf(codes.Internal, "validate session grant: %v", err)
	}

	if claims.CampaignID != campaignID {
		return aisessiongrant.Claims{}, status.Error(codes.PermissionDenied, "session grant does not match campaign turn target")
	}
	if sessionID != "" && claims.SessionID != sessionID {
		return aisessiongrant.Claims{}, status.Error(codes.PermissionDenied, "session grant does not match campaign turn target")
	}
	if agentID != "" && claims.AIAgentID != agentID {
		return aisessiongrant.Claims{}, status.Error(codes.PermissionDenied, "session grant does not match campaign turn target")
	}

	authState, err := s.getCampaignAIAuthState(ctx, campaignID, false)
	if err != nil {
		return aisessiongrant.Claims{}, err
	}
	if grantMatchesAuthState(claims, authState) {
		return claims, nil
	}

	// One forced refresh avoids false denials when projection updates race with grant issuance.
	refreshedState, err := s.getCampaignAIAuthState(ctx, campaignID, true)
	if err != nil {
		return aisessiongrant.Claims{}, err
	}
	if !grantMatchesAuthState(claims, refreshedState) {
		return aisessiongrant.Claims{}, status.Error(codes.FailedPrecondition, "session grant is stale")
	}
	return claims, nil
}

func (s *Service) validateSessionGrantConfig() error {
	if s == nil {
		return status.Error(codes.Internal, "service is not configured")
	}
	if strings.TrimSpace(s.sessionGrantConfig.Issuer) == "" ||
		strings.TrimSpace(s.sessionGrantConfig.Audience) == "" ||
		len(s.sessionGrantConfig.HMACKey) < 32 {
		return status.Error(codes.Internal, "session grant verifier is not configured")
	}
	return nil
}

func (s *Service) getCampaignAIAuthState(ctx context.Context, campaignID string, forceRefresh bool) (campaignAIAuthState, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignAIAuthState{}, status.Error(codes.InvalidArgument, "campaign_id is required")
	}
	if !forceRefresh {
		if cached, ok := s.campaignAuthStateCache.get(campaignID); ok {
			return cached, nil
		}
	}
	if s.gameCampaignAIClient == nil {
		return campaignAIAuthState{}, status.Error(codes.FailedPrecondition, "game campaign ai client is not configured")
	}

	resp, err := s.gameCampaignAIClient.GetCampaignAIAuthState(ctx, &gamev1.GetCampaignAIAuthStateRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return campaignAIAuthState{}, status.Errorf(codes.Internal, "get campaign ai auth state: %v", err)
	}
	state := campaignAIAuthState{
		CampaignID:      campaignID,
		AIAgentID:       strings.TrimSpace(resp.GetAiAgentId()),
		ActiveSessionID: strings.TrimSpace(resp.GetActiveSessionId()),
		AuthEpoch:       resp.GetAuthEpoch(),
		RefreshedAt:     s.clock().UTC(),
	}
	s.campaignAuthStateCache.put(state)
	return state, nil
}

func grantMatchesAuthState(claims aisessiongrant.Claims, state campaignAIAuthState) bool {
	return claims.AuthEpoch == state.AuthEpoch &&
		claims.SessionID == state.ActiveSessionID &&
		claims.AIAgentID == state.AIAgentID
}
