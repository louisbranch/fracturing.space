package ai

import (
	"context"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gamegrpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) validateCampaignContext(ctx context.Context, campaignID string, action gamev1.AuthorizationAction) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	if s.isAllowedInternalCampaignContextCaller(ctx) {
		return nil
	}

	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return status.Error(codes.PermissionDenied, "missing caller identity")
	}
	if s == nil || s.gameAuthorizationClient == nil {
		return status.Error(codes.FailedPrecondition, "campaign authorization client is unavailable")
	}

	authCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := s.gameAuthorizationClient.Can(authCtx, &gamev1.CanRequest{
		CampaignId: campaignID,
		Action:     action,
		Resource:   gamev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "authorize campaign context: %v", err)
	}
	if resp == nil || !resp.GetAllowed() {
		return status.Error(codes.PermissionDenied, "campaign access denied")
	}
	return nil
}

func (s *Service) isAllowedInternalCampaignContextCaller(ctx context.Context) bool {
	if s == nil || len(s.internalServiceAllowlist) == 0 {
		return false
	}
	serviceID := strings.ToLower(strings.TrimSpace(gamegrpcmeta.ServiceIDFromContext(ctx)))
	if serviceID == "" {
		return false
	}
	_, ok := s.internalServiceAllowlist[serviceID]
	return ok
}
