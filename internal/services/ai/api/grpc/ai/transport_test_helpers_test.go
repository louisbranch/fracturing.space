package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
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

func newTestInvocationHandlers(store *fakeStore) *InvocationHandlers {
	svc := newInvocationHandlersWithStores(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{}
	svc.providerInvocationAdapters[provider.OpenAI] = adapter
	return svc
}

func newTestAgentHandlers(store *fakeStore) *AgentHandlers {
	svc := newAgentHandlersWithStores(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{}
	svc.providerModelAdapters[provider.OpenAI] = adapter
	return svc
}

func newCredentialHandlersWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *CredentialHandlers {
	return NewCredentialHandlers(CredentialHandlersConfig{
		CredentialStore: credentialStore,
		AgentStore:      agentStore,
		Sealer:          sealer,
	})
}

func newProviderGrantHandlersWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *ProviderGrantHandlers {
	cfg := ProviderGrantHandlersConfig{
		AgentStore: agentStore,
		Sealer:     sealer,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		},
	}
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		cfg.ProviderGrantStore = store
	}
	if cfg.ProviderGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			cfg.ProviderGrantStore = store
		}
	}
	if store, ok := credentialStore.(storage.ProviderConnectSessionStore); ok {
		cfg.ConnectSessionStore = store
	}
	if cfg.ConnectSessionStore == nil {
		if store, ok := agentStore.(storage.ProviderConnectSessionStore); ok {
			cfg.ConnectSessionStore = store
		}
	}
	return NewProviderGrantHandlers(cfg)
}

func newAccessRequestHandlersWithStores(agentStore storage.AgentStore, accessRequestStore storage.AccessRequestStore, auditEventStore storage.AuditEventStore) *AccessRequestHandlers {
	return NewAccessRequestHandlers(AccessRequestHandlersConfig{
		AgentStore:         agentStore,
		AccessRequestStore: accessRequestStore,
		AuditEventStore:    auditEventStore,
	})
}

func newAgentHandlersConfigWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) AgentHandlersConfig {
	cfg := AgentHandlersConfig{
		CredentialStore: credentialStore,
		AgentStore:      agentStore,
		Sealer:          sealer,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		},
		ProviderModelAdapters: map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI: &fakeProviderInvocationAdapter{},
		},
	}
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		cfg.ProviderGrantStore = store
	}
	if cfg.ProviderGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			cfg.ProviderGrantStore = store
		}
	}
	if store, ok := credentialStore.(storage.AccessRequestStore); ok {
		cfg.AccessRequestStore = store
	}
	if cfg.AccessRequestStore == nil {
		if store, ok := agentStore.(storage.AccessRequestStore); ok {
			cfg.AccessRequestStore = store
		}
	}
	return cfg
}

func newAgentHandlersWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *AgentHandlers {
	return NewAgentHandlers(newAgentHandlersConfigWithStores(credentialStore, agentStore, sealer))
}

func newCampaignOrchestrationHandlersConfigWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) CampaignOrchestrationHandlersConfig {
	cfg := CampaignOrchestrationHandlersConfig{
		AgentStore:      agentStore,
		CredentialStore: credentialStore,
		Sealer:          sealer,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		},
		ProviderToolAdapters: map[provider.Provider]orchestration.Provider{
			provider.OpenAI: &fakeProviderToolAdapter{},
		},
	}
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		cfg.ProviderGrantStore = store
	}
	if cfg.ProviderGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			cfg.ProviderGrantStore = store
		}
	}
	return cfg
}

func newInvocationHandlersWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *InvocationHandlers {
	return NewInvocationHandlers(newInvocationHandlersConfigWithStores(credentialStore, agentStore, sealer))
}

func newInvocationHandlersConfigWithStores(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) InvocationHandlersConfig {
	cfg := InvocationHandlersConfig{
		CredentialStore: credentialStore,
		AgentStore:      agentStore,
		Sealer:          sealer,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		},
		ProviderInvocationAdapters: map[provider.Provider]provider.InvocationAdapter{
			provider.OpenAI: &fakeProviderInvocationAdapter{},
		},
	}
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		cfg.ProviderGrantStore = store
	}
	if cfg.ProviderGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			cfg.ProviderGrantStore = store
		}
	}
	if store, ok := credentialStore.(storage.AccessRequestStore); ok {
		cfg.AccessRequestStore = store
	}
	if cfg.AccessRequestStore == nil {
		if store, ok := agentStore.(storage.AccessRequestStore); ok {
			cfg.AccessRequestStore = store
		}
	}
	if store, ok := credentialStore.(storage.AuditEventStore); ok {
		cfg.AuditEventStore = store
	}
	if cfg.AuditEventStore == nil {
		if store, ok := agentStore.(storage.AuditEventStore); ok {
			cfg.AuditEventStore = store
		}
	}
	return cfg
}
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
