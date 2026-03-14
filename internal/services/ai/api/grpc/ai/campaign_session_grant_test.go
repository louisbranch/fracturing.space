package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakeCampaignAIAuthStateClient struct {
	states       []*gamev1.GetCampaignAIAuthStateResponse
	errs         []error
	calls        int
	lastReq      *gamev1.GetCampaignAIAuthStateRequest
	usageByAgent map[string]int32
	usageErr     error
	lastUsageReq *gamev1.GetCampaignAIBindingUsageRequest
}

func (f *fakeCampaignAIAuthStateClient) IssueCampaignAISessionGrant(context.Context, *gamev1.IssueCampaignAISessionGrantRequest, ...grpc.CallOption) (*gamev1.IssueCampaignAISessionGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeCampaignAIAuthStateClient) GetCampaignAIBindingUsage(_ context.Context, req *gamev1.GetCampaignAIBindingUsageRequest, _ ...grpc.CallOption) (*gamev1.GetCampaignAIBindingUsageResponse, error) {
	f.lastUsageReq = req
	if f.usageErr != nil {
		return nil, f.usageErr
	}
	if f.usageByAgent == nil {
		return &gamev1.GetCampaignAIBindingUsageResponse{}, nil
	}
	return &gamev1.GetCampaignAIBindingUsageResponse{
		ActiveCampaignCount: f.usageByAgent[req.GetAiAgentId()],
	}, nil
}

func (f *fakeCampaignAIAuthStateClient) GetCampaignAIAuthState(_ context.Context, req *gamev1.GetCampaignAIAuthStateRequest, _ ...grpc.CallOption) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	f.calls++
	f.lastReq = req
	idx := f.calls - 1
	if idx >= 0 && idx < len(f.errs) && f.errs[idx] != nil {
		return nil, f.errs[idx]
	}
	if idx >= 0 && idx < len(f.states) {
		return proto.Clone(f.states[idx]).(*gamev1.GetCampaignAIAuthStateResponse), nil
	}
	if len(f.states) > 0 {
		return proto.Clone(f.states[len(f.states)-1]).(*gamev1.GetCampaignAIAuthStateResponse), nil
	}
	return &gamev1.GetCampaignAIAuthStateResponse{}, nil
}

func testGrantConfig(now time.Time) aisessiongrant.Config {
	return aisessiongrant.Config{
		Issuer:   "test-issuer",
		Audience: "test-audience",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      10 * time.Minute,
		Now:      func() time.Time { return now },
	}
}

func newGrantValidationService(now time.Time, client gamev1.CampaignAIServiceClient, cfg aisessiongrant.Config) *Service {
	cache := newCampaignAIAuthStateCache()
	cache.now = func() time.Time { return now }
	return &Service{
		gameCampaignAIClient:   client,
		sessionGrantConfig:     cfg,
		clock:                  func() time.Time { return now },
		campaignAuthStateCache: cache,
	}
}

func TestValidateSessionGrantConfig(t *testing.T) {
	var nilService *Service
	assertStatusCode(t, nilService.validateSessionGrantConfig(), codes.Internal)

	svc := &Service{}
	assertStatusCode(t, svc.validateSessionGrantConfig(), codes.Internal)

	svc.sessionGrantConfig = testGrantConfig(time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC))
	if err := svc.validateSessionGrantConfig(); err != nil {
		t.Fatalf("validate session grant config: %v", err)
	}
}

func TestGetCampaignAIAuthStateUsesCache(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	client := &fakeCampaignAIAuthStateClient{
		states: []*gamev1.GetCampaignAIAuthStateResponse{{
			CampaignId:      "camp-1",
			AiAgentId:       "agent-1",
			ActiveSessionId: "session-1",
			AuthEpoch:       7,
		}},
	}
	svc := newGrantValidationService(now, client, testGrantConfig(now))

	state, err := svc.getCampaignAIAuthState(context.Background(), "camp-1", false)
	if err != nil {
		t.Fatalf("get campaign ai auth state: %v", err)
	}
	if state.CampaignID != "camp-1" || state.AIAgentID != "agent-1" || state.ActiveSessionID != "session-1" || state.AuthEpoch != 7 {
		t.Fatalf("unexpected state: %+v", state)
	}
	if !state.RefreshedAt.Equal(now.UTC()) {
		t.Fatalf("refreshed at = %s, want %s", state.RefreshedAt, now.UTC())
	}

	cached, err := svc.getCampaignAIAuthState(context.Background(), "camp-1", false)
	if err != nil {
		t.Fatalf("get cached campaign ai auth state: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("client calls = %d, want %d", client.calls, 1)
	}
	if cached != state {
		t.Fatalf("cached state = %+v, want %+v", cached, state)
	}
}

func TestGetCampaignAIAuthStateValidationAndErrorPaths(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	svc := newGrantValidationService(now, nil, testGrantConfig(now))
	_, err := svc.getCampaignAIAuthState(context.Background(), "", false)
	assertStatusCode(t, err, codes.InvalidArgument)
	_, err = svc.getCampaignAIAuthState(context.Background(), "camp-1", false)
	assertStatusCode(t, err, codes.FailedPrecondition)

	client := &fakeCampaignAIAuthStateClient{errs: []error{errors.New("boom")}}
	svc = newGrantValidationService(now, client, testGrantConfig(now))
	_, err = svc.getCampaignAIAuthState(context.Background(), "camp-1", false)
	assertStatusCode(t, err, codes.Internal)
}

func TestValidateCampaignSessionGrantRefreshesOnceOnMismatch(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	cfg := testGrantConfig(now)
	grantToken, claims, err := aisessiongrant.Issue(cfg, aisessiongrant.IssueInput{
		GrantID:    "grant-1",
		CampaignID: "camp-1",
		SessionID:  "session-1",
		AIAgentID:  "agent-1",
		AuthEpoch:  22,
	})
	if err != nil {
		t.Fatalf("issue test session grant: %v", err)
	}

	client := &fakeCampaignAIAuthStateClient{
		states: []*gamev1.GetCampaignAIAuthStateResponse{
			{CampaignId: "camp-1", AiAgentId: "agent-1", ActiveSessionId: "session-1", AuthEpoch: 21},
			{CampaignId: "camp-1", AiAgentId: "agent-1", ActiveSessionId: "session-1", AuthEpoch: 22},
		},
	}
	svc := newGrantValidationService(now, client, cfg)

	validated, err := svc.validateCampaignSessionGrant(context.Background(), grantToken, "camp-1", "session-1", "agent-1")
	if err != nil {
		t.Fatalf("validate campaign session grant: %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("client calls = %d, want %d", client.calls, 2)
	}
	if validated != claims {
		t.Fatalf("validated claims = %+v, want %+v", validated, claims)
	}
}

func TestValidateCampaignSessionGrantRejectsStaleAfterRefresh(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	cfg := testGrantConfig(now)
	grantToken, _, err := aisessiongrant.Issue(cfg, aisessiongrant.IssueInput{
		GrantID:    "grant-1",
		CampaignID: "camp-1",
		SessionID:  "session-1",
		AIAgentID:  "agent-1",
		AuthEpoch:  22,
	})
	if err != nil {
		t.Fatalf("issue test session grant: %v", err)
	}

	client := &fakeCampaignAIAuthStateClient{
		states: []*gamev1.GetCampaignAIAuthStateResponse{
			{CampaignId: "camp-1", AiAgentId: "agent-1", ActiveSessionId: "session-1", AuthEpoch: 21},
			{CampaignId: "camp-1", AiAgentId: "agent-1", ActiveSessionId: "session-1", AuthEpoch: 21},
		},
	}
	svc := newGrantValidationService(now, client, cfg)
	_, err = svc.validateCampaignSessionGrant(context.Background(), grantToken, "camp-1", "session-1", "agent-1")
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestValidateCampaignSessionGrantRejectsExpiredAndTargetMismatch(t *testing.T) {
	now := time.Date(2026, 3, 2, 6, 0, 0, 0, time.UTC)
	issueCfg := testGrantConfig(now)
	token, _, err := aisessiongrant.Issue(issueCfg, aisessiongrant.IssueInput{
		GrantID:    "grant-1",
		CampaignID: "camp-1",
		SessionID:  "session-1",
		AIAgentID:  "agent-1",
		AuthEpoch:  1,
	})
	if err != nil {
		t.Fatalf("issue test session grant: %v", err)
	}

	expiredCfg := testGrantConfig(now.Add(20 * time.Minute))
	expiredSvc := newGrantValidationService(now.Add(20*time.Minute), &fakeCampaignAIAuthStateClient{}, expiredCfg)
	_, err = expiredSvc.validateCampaignSessionGrant(context.Background(), token, "camp-1", "session-1", "agent-1")
	assertStatusCode(t, err, codes.PermissionDenied)

	matchSvc := newGrantValidationService(now, &fakeCampaignAIAuthStateClient{}, issueCfg)
	_, err = matchSvc.validateCampaignSessionGrant(context.Background(), token, "camp-2", "session-1", "agent-1")
	assertStatusCode(t, err, codes.PermissionDenied)
	_, err = matchSvc.validateCampaignSessionGrant(context.Background(), token, "camp-1", "session-2", "agent-1")
	assertStatusCode(t, err, codes.PermissionDenied)
	_, err = matchSvc.validateCampaignSessionGrant(context.Background(), token, "camp-1", "session-1", "agent-2")
	assertStatusCode(t, err, codes.PermissionDenied)
}
