package ai

import (
	"context"
	"errors"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/gamebridge"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CampaignAccessAuthorizer authorizes one caller against one campaign.
type CampaignAccessAuthorizer interface {
	AuthorizeCampaign(context.Context, string, string, gamev1.AuthorizationAction) error
	IsAllowedInternalServiceCaller(context.Context) bool
}

type campaignContextValidator struct {
	authorizer CampaignAccessAuthorizer
}

func newCampaignContextValidator(authorizer CampaignAccessAuthorizer) campaignContextValidator {
	return campaignContextValidator{
		authorizer: authorizer,
	}
}

func (v campaignContextValidator) validateCampaignContext(ctx context.Context, campaignID string, action gamev1.AuthorizationAction) error {
	if campaignID == "" {
		return status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	if v.isAllowedInternalCampaignContextCaller(ctx) {
		return nil
	}

	userID, err := requireCallerUserID(ctx)
	if err != nil {
		return status.Error(codes.PermissionDenied, "missing caller identity")
	}
	if v.authorizer == nil {
		return status.Error(codes.FailedPrecondition, "campaign authorization client is unavailable")
	}
	if err := v.authorizer.AuthorizeCampaign(ctx, userID, campaignID, action); err != nil {
		switch {
		case errors.Is(err, gamebridge.ErrCampaignAuthorizationUnavailable):
			return status.Error(codes.FailedPrecondition, "campaign authorization client is unavailable")
		case errors.Is(err, gamebridge.ErrCampaignAccessDenied):
			return status.Error(codes.PermissionDenied, "campaign access denied")
		case errors.Is(err, gamebridge.ErrMissingCallerIdentity):
			return status.Error(codes.PermissionDenied, "missing caller identity")
		default:
			return transportErrorToStatus(err, transportErrorConfig{Operation: "authorize campaign context"})
		}
	}
	return nil
}

func (v campaignContextValidator) isAllowedInternalCampaignContextCaller(ctx context.Context) bool {
	return v.authorizer != nil && v.authorizer.IsAllowedInternalServiceCaller(ctx)
}
