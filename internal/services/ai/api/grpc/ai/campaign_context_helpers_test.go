package ai

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeGameAuthorizationClient struct {
	canResp *gamev1.CanResponse
	canErr  error

	lastCtx context.Context
	lastReq *gamev1.CanRequest
}

func (f *fakeGameAuthorizationClient) Can(ctx context.Context, req *gamev1.CanRequest, _ ...grpc.CallOption) (*gamev1.CanResponse, error) {
	f.lastCtx = ctx
	f.lastReq = req
	if f.canErr != nil {
		return nil, f.canErr
	}
	if f.canResp == nil {
		return &gamev1.CanResponse{Allowed: true}, nil
	}
	return f.canResp, nil
}

func (f *fakeGameAuthorizationClient) BatchCan(context.Context, *gamev1.BatchCanRequest, ...grpc.CallOption) (*gamev1.BatchCanResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestValidateCampaignContextForwardsUserIDToAuthorization(t *testing.T) {
	authz := &fakeGameAuthorizationClient{canResp: &gamev1.CanResponse{Allowed: true}}
	validator := newCampaignContextValidator(authz, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-123"))

	if err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		t.Fatalf("validateCampaignContext() error = %v", err)
	}
	if authz.lastReq == nil {
		t.Fatal("expected authorization request")
	}
	if authz.lastReq.GetCampaignId() != "campaign-1" {
		t.Fatalf("campaign_id = %q, want %q", authz.lastReq.GetCampaignId(), "campaign-1")
	}
	if authz.lastReq.GetAction() != gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ {
		t.Fatalf("action = %v, want %v", authz.lastReq.GetAction(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ)
	}

	outgoingMD, ok := metadata.FromOutgoingContext(authz.lastCtx)
	if !ok {
		t.Fatal("expected outgoing metadata on authorization request context")
	}
	userIDs := outgoingMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) == 0 || userIDs[0] != "user-123" {
		t.Fatalf("outgoing user ids = %v, want [user-123]", userIDs)
	}
}

func TestValidateCampaignContextAllowsConfiguredInternalService(t *testing.T) {
	validator := newCampaignContextValidator(nil, map[string]struct{}{"worker": {}})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))

	if err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		t.Fatalf("validateCampaignContext() error = %v", err)
	}
}

func TestValidateCampaignContextRejectsMissingCallerIdentity(t *testing.T) {
	validator := newCampaignContextValidator(nil, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "web"))

	err := validator.validateCampaignContext(ctx, "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.PermissionDenied, err)
	}
}
