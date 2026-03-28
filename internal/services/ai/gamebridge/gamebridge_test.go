package gamebridge

import (
	"context"
	"errors"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeAuthorizationClient struct {
	canResp *gamev1.CanResponse
	canErr  error

	lastCtx context.Context
	lastReq *gamev1.CanRequest
}

func (f *fakeAuthorizationClient) Can(ctx context.Context, req *gamev1.CanRequest, _ ...grpc.CallOption) (*gamev1.CanResponse, error) {
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

func (f *fakeAuthorizationClient) BatchCan(context.Context, *gamev1.BatchCanRequest, ...grpc.CallOption) (*gamev1.BatchCanResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type fakeCampaignAIClient struct {
	usageResp *gamev1.GetCampaignAIBindingUsageResponse
	usageErr  error
	authResp  *gamev1.GetCampaignAIAuthStateResponse
	authErr   error
}

func (f *fakeCampaignAIClient) IssueCampaignAISessionGrant(context.Context, *gamev1.IssueCampaignAISessionGrantRequest, ...grpc.CallOption) (*gamev1.IssueCampaignAISessionGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeCampaignAIClient) GetCampaignAIBindingUsage(context.Context, *gamev1.GetCampaignAIBindingUsageRequest, ...grpc.CallOption) (*gamev1.GetCampaignAIBindingUsageResponse, error) {
	if f.usageErr != nil {
		return nil, f.usageErr
	}
	if f.usageResp == nil {
		return &gamev1.GetCampaignAIBindingUsageResponse{}, nil
	}
	return f.usageResp, nil
}

func (f *fakeCampaignAIClient) GetCampaignAIAuthState(context.Context, *gamev1.GetCampaignAIAuthStateRequest, ...grpc.CallOption) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	if f.authErr != nil {
		return nil, f.authErr
	}
	if f.authResp == nil {
		return &gamev1.GetCampaignAIAuthStateResponse{}, nil
	}
	return f.authResp, nil
}

func TestAuthorizeCampaignForwardsUserIDViaOutgoingMetadata(t *testing.T) {
	authz := &fakeAuthorizationClient{canResp: &gamev1.CanResponse{Allowed: true}}
	gateway := New(Config{Authorization: authz})

	if err := gateway.AuthorizeCampaign(context.Background(), "user-123", "campaign-1", gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		t.Fatalf("AuthorizeCampaign() error = %v", err)
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

func TestIsAllowedInternalServiceCallerHonorsAllowlist(t *testing.T) {
	gateway := New(Config{
		InternalServiceAllowlist: map[string]struct{}{"Worker": {}},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))

	if !gateway.IsAllowedInternalServiceCaller(ctx) {
		t.Fatal("expected worker to be allowed")
	}
}

func TestActiveCampaignCountDegradesWithoutClient(t *testing.T) {
	gateway := New(Config{})

	count, err := gateway.ActiveCampaignCount(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("ActiveCampaignCount() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestCampaignAuthStateErrorsWithoutClient(t *testing.T) {
	gateway := New(Config{})

	_, err := gateway.CampaignAuthState(context.Background(), "campaign-1")
	if !errors.Is(err, ErrCampaignAuthStateUnavailable) {
		t.Fatalf("CampaignAuthState() error = %v, want ErrCampaignAuthStateUnavailable", err)
	}
}
