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

type campaignContextValidator struct {
	authorizationClient      gamev1.AuthorizationServiceClient
	internalServiceAllowlist map[string]struct{}
}

func newCampaignContextValidator(client gamev1.AuthorizationServiceClient, allowlist map[string]struct{}) campaignContextValidator {
	copiedAllowlist := make(map[string]struct{}, len(allowlist))
	for serviceID := range allowlist {
		copiedAllowlist[serviceID] = struct{}{}
	}
	return campaignContextValidator{
		authorizationClient:      client,
		internalServiceAllowlist: copiedAllowlist,
	}
}

func (v campaignContextValidator) validateCampaignContext(ctx context.Context, campaignID string, action gamev1.AuthorizationAction) error {
	if campaignID == "" {
		return status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	if v.isAllowedInternalCampaignContextCaller(ctx) {
		return nil
	}

	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return status.Error(codes.PermissionDenied, "missing caller identity")
	}
	if v.authorizationClient == nil {
		return status.Error(codes.FailedPrecondition, "campaign authorization client is unavailable")
	}

	authCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := v.authorizationClient.Can(authCtx, &gamev1.CanRequest{
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

func (v campaignContextValidator) isAllowedInternalCampaignContextCaller(ctx context.Context) bool {
	if len(v.internalServiceAllowlist) == 0 {
		return false
	}
	serviceID := strings.ToLower(strings.TrimSpace(gamegrpcmeta.ServiceIDFromContext(ctx)))
	if serviceID == "" {
		return false
	}
	_, ok := v.internalServiceAllowlist[serviceID]
	return ok
}
