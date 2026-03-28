package ai

import (
	"context"
	"errors"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/ai/gamebridge"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeCampaignAuthorizer struct {
	allowed              bool
	err                  error
	allowInternalService bool

	lastCtx      context.Context
	lastUserID   string
	lastCampaign string
	lastAction   gamev1.AuthorizationAction
}

func (f *fakeCampaignAuthorizer) AuthorizeCampaign(ctx context.Context, userID, campaignID string, action gamev1.AuthorizationAction) error {
	f.lastCtx = ctx
	f.lastUserID = userID
	f.lastCampaign = campaignID
	f.lastAction = action
	if f.err != nil {
		return f.err
	}
	if f.allowed {
		return nil
	}
	return gamebridge.ErrCampaignAccessDenied
}

func (f *fakeCampaignAuthorizer) IsAllowedInternalServiceCaller(context.Context) bool {
	return f.allowInternalService
}

func TestValidateCampaignContextCallsAuthorizerWithUserAndAction(t *testing.T) {
	authz := &fakeCampaignAuthorizer{allowed: true}
	validator := newCampaignContextValidator(authz)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-123"))

	if err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		t.Fatalf("validateCampaignContext() error = %v", err)
	}
	if authz.lastUserID != "user-123" {
		t.Fatalf("user_id = %q, want %q", authz.lastUserID, "user-123")
	}
	if authz.lastCampaign != "campaign-1" {
		t.Fatalf("campaign_id = %q, want %q", authz.lastCampaign, "campaign-1")
	}
	if authz.lastAction != gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ {
		t.Fatalf("action = %v, want %v", authz.lastAction, gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ)
	}
}

func TestValidateCampaignContextAllowsConfiguredInternalService(t *testing.T) {
	validator := newCampaignContextValidator(&fakeCampaignAuthorizer{allowInternalService: true})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))

	if err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		t.Fatalf("validateCampaignContext() error = %v", err)
	}
}

func TestValidateCampaignContextRejectsMissingCallerIdentity(t *testing.T) {
	validator := newCampaignContextValidator(&fakeCampaignAuthorizer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "web"))

	err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.PermissionDenied, err)
	}
}

func TestValidateCampaignContextMapsGatewayErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "authorization unavailable", err: gamebridge.ErrCampaignAuthorizationUnavailable, want: codes.FailedPrecondition},
		{name: "access denied", err: gamebridge.ErrCampaignAccessDenied, want: codes.PermissionDenied},
		{name: "missing caller identity", err: gamebridge.ErrMissingCallerIdentity, want: codes.PermissionDenied},
		{name: "internal", err: errors.New("boom"), want: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := newCampaignContextValidator(&fakeCampaignAuthorizer{err: tt.err})
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-123"))

			err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ)
			if status.Code(err) != tt.want {
				t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), tt.want, err)
			}
		})
	}
}
