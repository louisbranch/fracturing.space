package ai

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSealer = aifakes.Sealer

type fakeStore struct {
	*aifakes.CredentialStore
	*aifakes.AgentStore
	*aifakes.AccessRequestStore
	*aifakes.ProviderGrantStore
	*aifakes.ProviderConnectSessionStore
	*aifakes.CampaignArtifactStore
	*aifakes.AuditEventStore
}

// ListAccessibleAgents overrides the embedded AgentStore method to include
// agents reachable via approved invoke access requests.
func (s *fakeStore) ListAccessibleAgents(_ context.Context, userID string, pageSize int, pageToken string) (agent.Page, error) {
	seen := make(map[string]struct{})
	items := make([]agent.Agent, 0)

	// Owned agents.
	for _, rec := range s.Agents {
		if rec.OwnerUserID == userID {
			items = append(items, rec)
			seen[rec.ID] = struct{}{}
		}
	}

	// Shared agents via approved invoke access requests.
	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID != userID || ar.Scope != "invoke" || ar.Status != "approved" {
			continue
		}
		if _, ok := seen[ar.AgentID]; ok {
			continue
		}
		a, ok := s.Agents[ar.AgentID]
		if !ok || a.OwnerUserID != ar.OwnerUserID {
			continue
		}
		items = append(items, a)
		seen[ar.AgentID] = struct{}{}
	}

	// Sort by ID for deterministic pagination.
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	// Apply keyset pagination.
	start := 0
	if pageToken != "" {
		for i, rec := range items {
			if rec.ID > pageToken {
				start = i
				break
			}
			if i == len(items)-1 {
				start = len(items)
			}
		}
	}
	items = items[start:]

	if pageSize > 0 && len(items) > pageSize {
		nextToken := items[pageSize-1].ID
		return agent.Page{Agents: items[:pageSize], NextPageToken: nextToken}, nil
	}
	return agent.Page{Agents: items}, nil
}

type fakeCampaignAIAuthStateClient struct {
	usageByAgent map[string]int32
	usageErr     error
	authState    *gamev1.GetCampaignAIAuthStateResponse
	authStateErr error
}

func (f *fakeCampaignAIAuthStateClient) IssueCampaignAISessionGrant(context.Context, *gamev1.IssueCampaignAISessionGrantRequest, ...grpc.CallOption) (*gamev1.IssueCampaignAISessionGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeCampaignAIAuthStateClient) GetCampaignAIBindingUsage(_ context.Context, in *gamev1.GetCampaignAIBindingUsageRequest, _ ...grpc.CallOption) (*gamev1.GetCampaignAIBindingUsageResponse, error) {
	if f.usageErr != nil {
		return nil, f.usageErr
	}
	return &gamev1.GetCampaignAIBindingUsageResponse{
		ActiveCampaignCount: f.usageByAgent[in.GetAiAgentId()],
	}, nil
}

func (f *fakeCampaignAIAuthStateClient) GetCampaignAIAuthState(context.Context, *gamev1.GetCampaignAIAuthStateRequest, ...grpc.CallOption) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	if f.authStateErr != nil {
		return nil, f.authStateErr
	}
	if f.authState == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	return f.authState, nil
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		CredentialStore:             aifakes.NewCredentialStore(),
		AgentStore:                  aifakes.NewAgentStore(),
		AccessRequestStore:          aifakes.NewAccessRequestStore(),
		ProviderGrantStore:          aifakes.NewProviderGrantStore(),
		ProviderConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		CampaignArtifactStore:       aifakes.NewCampaignArtifactStore(),
		AuditEventStore:             aifakes.NewAuditEventStore(),
	}
}

type fakeProviderOAuthAdapter struct {
	buildAuthorizationURLErr error
	exchangeErr              error
	exchangeResult           provider.TokenExchangeResult
	refreshErr               error
	refreshResult            provider.TokenExchangeResult
	revokeErr                error

	lastAuthorizationInput provider.AuthorizationURLInput
	lastRefreshToken       string
	lastRevokedToken       string
}

func (f *fakeProviderOAuthAdapter) BuildAuthorizationURL(input provider.AuthorizationURLInput) (string, error) {
	f.lastAuthorizationInput = input
	if f.buildAuthorizationURLErr != nil {
		return "", f.buildAuthorizationURLErr
	}
	return "https://provider.example.com/auth", nil
}

func (f *fakeProviderOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, _ provider.AuthorizationCodeInput) (provider.TokenExchangeResult, error) {
	if f.exchangeErr != nil {
		return provider.TokenExchangeResult{}, f.exchangeErr
	}
	if strings.TrimSpace(f.exchangeResult.TokenPlaintext) == "" {
		return provider.TokenExchangeResult{TokenPlaintext: `{"access_token":"at-1","refresh_token":"rt-1"}`, RefreshSupported: true}, nil
	}
	return f.exchangeResult, nil
}

func (f *fakeProviderOAuthAdapter) RefreshToken(_ context.Context, input provider.RefreshTokenInput) (provider.TokenExchangeResult, error) {
	f.lastRefreshToken = input.RefreshToken
	if f.refreshErr != nil {
		return provider.TokenExchangeResult{}, f.refreshErr
	}
	return f.refreshResult, nil
}

func (f *fakeProviderOAuthAdapter) RevokeToken(_ context.Context, input provider.RevokeTokenInput) error {
	f.lastRevokedToken = input.Token
	return f.revokeErr
}

type defaultProviderOAuthAdapterForTests struct{}

func (d *defaultProviderOAuthAdapterForTests) BuildAuthorizationURL(input provider.AuthorizationURLInput) (string, error) {
	return "https://oauth.fracturing.space/openai?state=" + strings.TrimSpace(input.State), nil
}

func (d *defaultProviderOAuthAdapterForTests) ExchangeAuthorizationCode(_ context.Context, input provider.AuthorizationCodeInput) (provider.TokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return provider.TokenExchangeResult{}, errors.New("authorization code is required")
	}
	return provider.TokenExchangeResult{
		TokenPlaintext:   "token:" + code,
		RefreshSupported: true,
	}, nil
}

func (d *defaultProviderOAuthAdapterForTests) RefreshToken(_ context.Context, input provider.RefreshTokenInput) (provider.TokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return provider.TokenExchangeResult{}, errors.New("refresh token is required")
	}
	return provider.TokenExchangeResult{
		TokenPlaintext:   "token:refresh:" + refreshToken,
		RefreshSupported: true,
	}, nil
}

func (d *defaultProviderOAuthAdapterForTests) RevokeToken(_ context.Context, input provider.RevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return errors.New("token is required")
	}
	return nil
}

type fakeProviderInvocationAdapter struct {
	invokeErr           error
	invokeResult        provider.InvokeResult
	lastInput           provider.InvokeInput
	listModelsErr       error
	listModelsResult    []provider.Model
	lastListModelsInput provider.ListModelsInput
}

func (f *fakeProviderInvocationAdapter) Invoke(_ context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	f.lastInput = input
	if f.invokeErr != nil {
		return provider.InvokeResult{}, f.invokeErr
	}
	return f.invokeResult, nil
}

func (f *fakeProviderInvocationAdapter) ListModels(_ context.Context, input provider.ListModelsInput) ([]provider.Model, error) {
	f.lastListModelsInput = input
	if f.listModelsErr != nil {
		return nil, f.listModelsErr
	}
	if f.listModelsResult == nil {
		return []provider.Model{
			{ID: "gpt-4o-mini", OwnedBy: "openai"},
			{ID: "gpt-4o", OwnedBy: "openai"},
		}, nil
	}
	return f.listModelsResult, nil
}

type fakeCampaignTurnRunner struct {
	runErr    error
	runResult orchestration.Result
	lastInput orchestration.Input
}

func (f *fakeCampaignTurnRunner) Run(_ context.Context, input orchestration.Input) (orchestration.Result, error) {
	f.lastInput = input
	if f.runErr != nil {
		return orchestration.Result{}, f.runErr
	}
	return f.runResult, nil
}

type fakeProviderToolAdapter struct {
	runErr    error
	runResult orchestration.ProviderOutput
	lastInput orchestration.ProviderInput
}

func (f *fakeProviderToolAdapter) Run(_ context.Context, input orchestration.ProviderInput) (orchestration.ProviderOutput, error) {
	f.lastInput = input
	if f.runErr != nil {
		return orchestration.ProviderOutput{}, f.runErr
	}
	return f.runResult, nil
}

// --- Agent handler test helpers ---

type agentTestOpts struct {
	clock                service.Clock
	idGenerator          service.IDGenerator
	oauthAdapters        map[provider.Provider]provider.OAuthAdapter
	modelAdapters        map[provider.Provider]provider.ModelAdapter
	gameCampaignAIClient gamev1.CampaignAIServiceClient
}

func newAgentHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *AgentHandlers {
	t.Helper()
	return newAgentHandlersWithOpts(t, credentialStore, agentStore, sealer, agentTestOpts{})
}

func newAgentHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts agentTestOpts) *AgentHandlers {
	t.Helper()

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		}
	}
	modelAdapters := opts.modelAdapters
	if modelAdapters == nil {
		modelAdapters = map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: &fakeProviderInvocationAdapter{},
		}
	}

	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}
	var accessRequestStore storage.AccessRequestStore
	if store, ok := credentialStore.(storage.AccessRequestStore); ok {
		accessRequestStore = store
	}
	if accessRequestStore == nil {
		if store, ok := agentStore.(storage.AccessRequestStore); ok {
			accessRequestStore = store
		}
	}

	authTokenResolver := service.NewAuthTokenResolver(service.AuthTokenResolverConfig{
		CredentialStore:       credentialStore,
		ProviderGrantStore:    providerGrantStore,
		ProviderOAuthAdapters: oauthAdapters,
		Sealer:                sealer,
		Clock:                 opts.clock,
	})
	accessibleAgentResolver := service.NewAccessibleAgentResolver(agentStore, accessRequestStore)

	var usageGuard *service.UsageGuard
	if opts.gameCampaignAIClient != nil {
		usageGuard = service.NewUsageGuard(agentStore, opts.gameCampaignAIClient)
	}

	agentSvc, err := service.NewAgentService(service.AgentServiceConfig{
		CredentialStore:         credentialStore,
		AgentStore:              agentStore,
		ProviderGrantStore:      providerGrantStore,
		AccessRequestStore:      accessRequestStore,
		ProviderModelAdapters:   modelAdapters,
		AuthTokenResolver:       authTokenResolver,
		AccessibleAgentResolver: accessibleAgentResolver,
		UsageGuard:              usageGuard,
		Clock:                   opts.clock,
		IDGenerator:             opts.idGenerator,
	})
	if err != nil {
		t.Fatalf("NewAgentService: %v", err)
	}
	h, err := NewAgentHandlers(AgentHandlersConfig{
		AgentService: agentSvc,
	})
	if err != nil {
		t.Fatalf("NewAgentHandlers: %v", err)
	}
	return h
}

func newTestAgentHandlers(t *testing.T, store *fakeStore) *AgentHandlers {
	t.Helper()
	return newAgentHandlersWithStores(t, store, store, &fakeSealer{})
}

// --- Invocation handler test helpers ---

type invocationTestOpts struct {
	clock              service.Clock
	oauthAdapters      map[provider.Provider]provider.OAuthAdapter
	invocationAdapters map[provider.Provider]provider.InvocationAdapter
}

func newInvocationHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *InvocationHandlers {
	t.Helper()
	return newInvocationHandlersWithOpts(t, credentialStore, agentStore, sealer, invocationTestOpts{})
}

func newInvocationHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts invocationTestOpts) *InvocationHandlers {
	t.Helper()

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		}
	}
	invocationAdapters := opts.invocationAdapters
	if invocationAdapters == nil {
		invocationAdapters = map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeProviderInvocationAdapter{},
		}
	}

	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}
	var accessRequestStore storage.AccessRequestStore
	if store, ok := credentialStore.(storage.AccessRequestStore); ok {
		accessRequestStore = store
	}
	if accessRequestStore == nil {
		if store, ok := agentStore.(storage.AccessRequestStore); ok {
			accessRequestStore = store
		}
	}
	var auditEventStore storage.AuditEventStore
	if store, ok := credentialStore.(storage.AuditEventStore); ok {
		auditEventStore = store
	}
	if auditEventStore == nil {
		if store, ok := agentStore.(storage.AuditEventStore); ok {
			auditEventStore = store
		}
	}

	authTokenResolver := service.NewAuthTokenResolver(service.AuthTokenResolverConfig{
		CredentialStore:       credentialStore,
		ProviderGrantStore:    providerGrantStore,
		ProviderOAuthAdapters: oauthAdapters,
		Sealer:                sealer,
		Clock:                 opts.clock,
	})
	accessibleAgentResolver := service.NewAccessibleAgentResolver(agentStore, accessRequestStore)

	invocationSvc, err := service.NewInvocationService(service.InvocationServiceConfig{
		AgentStore:                 agentStore,
		AuditEventStore:            auditEventStore,
		AccessibleAgentResolver:    accessibleAgentResolver,
		AuthTokenResolver:          authTokenResolver,
		ProviderInvocationAdapters: invocationAdapters,
		Clock:                      opts.clock,
	})
	if err != nil {
		t.Fatalf("NewInvocationService: %v", err)
	}
	h, err := NewInvocationHandlers(InvocationHandlersConfig{
		InvocationService: invocationSvc,
	})
	if err != nil {
		t.Fatalf("NewInvocationHandlers: %v", err)
	}
	return h
}

// --- Campaign orchestration handler test helpers ---

type campaignOrchestrationTestOpts struct {
	clock                service.Clock
	oauthAdapters        map[provider.Provider]provider.OAuthAdapter
	toolAdapters         map[provider.Provider]orchestration.Provider
	campaignTurnRunner   orchestration.CampaignTurnRunner
	sessionGrantConfig   *aisessiongrant.Config
	gameCampaignAIClient gamev1.CampaignAIServiceClient
}

func newCampaignOrchestrationHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts campaignOrchestrationTestOpts) *CampaignOrchestrationHandlers {
	t.Helper()

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		}
	}
	toolAdapters := opts.toolAdapters
	if toolAdapters == nil {
		toolAdapters = map[provider.Provider]orchestration.Provider{
			provider.OpenAI: &fakeProviderToolAdapter{},
		}
	}

	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}

	authTokenResolver := service.NewAuthTokenResolver(service.AuthTokenResolverConfig{
		CredentialStore:       credentialStore,
		ProviderGrantStore:    providerGrantStore,
		ProviderOAuthAdapters: oauthAdapters,
		Sealer:                sealer,
		Clock:                 opts.clock,
	})

	orchestrationSvc, err := service.NewCampaignOrchestrationService(service.CampaignOrchestrationServiceConfig{
		AgentStore:           agentStore,
		GameCampaignAIClient: opts.gameCampaignAIClient,
		ProviderToolAdapters: toolAdapters,
		CampaignTurnRunner:   opts.campaignTurnRunner,
		SessionGrantConfig:   opts.sessionGrantConfig,
		AuthTokenResolver:    authTokenResolver,
	})
	if err != nil {
		t.Fatalf("NewCampaignOrchestrationService: %v", err)
	}
	h, err := NewCampaignOrchestrationHandlers(CampaignOrchestrationHandlersConfig{
		CampaignOrchestrationService: orchestrationSvc,
	})
	if err != nil {
		t.Fatalf("NewCampaignOrchestrationHandlers: %v", err)
	}
	return h
}

// --- Credential handler test helpers ---

func newCredentialHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *CredentialHandlers {
	t.Helper()
	return newCredentialHandlersWithOpts(t, credentialStore, agentStore, sealer, nil, nil, nil)
}

func newCredentialHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, clock service.Clock, idGen service.IDGenerator, usageGuard *service.UsageGuard) *CredentialHandlers {
	t.Helper()
	if usageGuard == nil {
		usageGuard = service.NewUsageGuard(agentStore, nil)
	}
	credSvc, err := service.NewCredentialService(service.CredentialServiceConfig{
		CredentialStore: credentialStore,
		Sealer:          sealer,
		UsageGuard:      usageGuard,
		Clock:           clock,
		IDGenerator:     idGen,
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}
	h, err := NewCredentialHandlers(CredentialHandlersConfig{
		CredentialService: credSvc,
	})
	if err != nil {
		t.Fatalf("NewCredentialHandlers: %v", err)
	}
	return h
}

// --- Provider grant handler test helpers ---

// providerGrantTestHandlers bundles the service and handler so tests can
// configure clock/ID generator on the service while calling RPC methods on
// the handler.
type providerGrantTestHandlers struct {
	*ProviderGrantHandlers
	svc *service.ProviderGrantService
}

type providerGrantTestOpts struct {
	clock                 service.Clock
	idGenerator           service.IDGenerator
	codeVerifierGenerator service.CodeVerifierGenerator
	oauthAdapters         map[provider.Provider]provider.OAuthAdapter
	usageGuard            *service.UsageGuard
}

func newProviderGrantHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *providerGrantTestHandlers {
	t.Helper()
	return newProviderGrantHandlersWithOpts(t, credentialStore, agentStore, sealer, providerGrantTestOpts{})
}

func newProviderGrantHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts providerGrantTestOpts) *providerGrantTestHandlers {
	t.Helper()

	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}
	var connectSessionStore storage.ProviderConnectSessionStore
	if store, ok := credentialStore.(storage.ProviderConnectSessionStore); ok {
		connectSessionStore = store
	}
	if connectSessionStore == nil {
		if store, ok := agentStore.(storage.ProviderConnectSessionStore); ok {
			connectSessionStore = store
		}
	}

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		}
	}
	usageGuard := opts.usageGuard
	if usageGuard == nil {
		usageGuard = service.NewUsageGuard(agentStore, nil)
	}

	svc, err := service.NewProviderGrantService(service.ProviderGrantServiceConfig{
		ProviderGrantStore:    providerGrantStore,
		ConnectSessionStore:   connectSessionStore,
		Sealer:                sealer,
		ProviderOAuthAdapters: oauthAdapters,
		UsageGuard:            usageGuard,
		Clock:                 opts.clock,
		IDGenerator:           opts.idGenerator,
		CodeVerifierGenerator: opts.codeVerifierGenerator,
	})
	if err != nil {
		t.Fatalf("NewProviderGrantService: %v", err)
	}
	h, err := NewProviderGrantHandlers(ProviderGrantHandlersConfig{
		ProviderGrantService: svc,
	})
	if err != nil {
		t.Fatalf("NewProviderGrantHandlers: %v", err)
	}
	return &providerGrantTestHandlers{ProviderGrantHandlers: h, svc: svc}
}

// --- Access request handler test helpers ---

// accessRequestTestHandlers bundles the service and handler so tests can
// configure clock/ID generator on the service while calling RPC methods on
// the handler.
type accessRequestTestHandlers struct {
	*AccessRequestHandlers
	svc *service.AccessRequestService
}

func newAccessRequestHandlersWithStores(t *testing.T, agentStore storage.AgentStore, accessRequestStore storage.AccessRequestStore, auditEventStore storage.AuditEventStore) *accessRequestTestHandlers {
	t.Helper()
	svc, err := service.NewAccessRequestService(service.AccessRequestServiceConfig{
		AgentStore:         agentStore,
		AccessRequestStore: accessRequestStore,
		AuditEventStore:    auditEventStore,
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}
	h, err := NewAccessRequestHandlers(AccessRequestHandlersConfig{
		AccessRequestService: svc,
	})
	if err != nil {
		t.Fatalf("NewAccessRequestHandlers: %v", err)
	}
	return &accessRequestTestHandlers{AccessRequestHandlers: h, svc: svc}
}

// --- Common test helpers ---

func ptrTime(value time.Time) *time.Time {
	return &value
}

func testAISessionGrantConfig() aisessiongrant.Config {
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	return aisessiongrant.Config{
		Issuer:   "fracturing-space-game",
		Audience: "fracturing-space-ai",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      10 * time.Minute,
		Now: func() time.Time {
			return now
		},
	}
}

func mustIssueAISessionGrant(t *testing.T, cfg aisessiongrant.Config, input aisessiongrant.IssueInput) string {
	t.Helper()
	token, _, err := aisessiongrant.Issue(cfg, input)
	if err != nil {
		t.Fatalf("issue ai session grant: %v", err)
	}
	return token
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected status %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != want {
		t.Fatalf("expected status %v, got %v", want, st.Code())
	}
}

// pkceCodeChallengeS256 computes the S256 code challenge for test assertions.
func pkceCodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func assertStatusReason(t *testing.T, err error, want apperrors.Code) {
	t.Helper()
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			if info.Reason != string(want) {
				t.Fatalf("expected reason %q, got %q", want, info.Reason)
			}
			return
		}
	}
	t.Fatalf("missing ErrorInfo detail in %v", err)
}
