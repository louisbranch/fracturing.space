package ai

import (
	"context"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

const (
	// userIDHeader is injected by trusted edge/auth layers and consumed here for
	// ownership enforcement. Direct callers must not be allowed to spoof it.
	userIDHeader = "x-fracturing-space-user-id"

	defaultPageSize = 10
	maxPageSize     = 50

	providerGrantRefreshWindow = 2 * time.Minute
)

// SecretSealer encrypts secret values before persistence.
type SecretSealer interface {
	Seal(value string) (string, error)
	Open(sealed string) (string, error)
}

// ProviderOAuthAdapter handles provider-specific OAuth URL/token exchange logic.
type ProviderOAuthAdapter interface {
	BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error)
	ExchangeAuthorizationCode(ctx context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error)
	RefreshToken(ctx context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error)
	RevokeToken(ctx context.Context, input ProviderRevokeTokenInput) error
}

// ProviderAuthorizationURLInput contains parameters for building provider auth URL.
type ProviderAuthorizationURLInput struct {
	State           string
	CodeChallenge   string
	RequestedScopes []string
}

// ProviderAuthorizationCodeInput contains token-exchange input fields.
type ProviderAuthorizationCodeInput struct {
	AuthorizationCode string
	CodeVerifier      string
}

// ProviderRefreshTokenInput contains refresh-token input fields.
type ProviderRefreshTokenInput struct {
	RefreshToken string
}

// ProviderRevokeTokenInput contains token-revocation input fields.
type ProviderRevokeTokenInput struct {
	Token string
}

// ProviderTokenExchangeResult contains provider token exchange output.
type ProviderTokenExchangeResult struct {
	TokenPlaintext   string
	RefreshSupported bool
	ExpiresAt        *time.Time
	LastRefreshError string
}

// ProviderInvocationAdapter handles provider-specific inference invocation.
type ProviderInvocationAdapter interface {
	Invoke(ctx context.Context, input ProviderInvokeInput) (ProviderInvokeResult, error)
}

// ProviderModelAdapter handles provider-backed model discovery.
type ProviderModelAdapter interface {
	ListModels(ctx context.Context, input ProviderListModelsInput) ([]ProviderModel, error)
}

// ProviderInvokeInput contains provider invocation input fields.
type ProviderInvokeInput struct {
	Model        string
	Input        string
	Instructions string
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// ProviderInvokeResult contains invocation output.
type ProviderInvokeResult struct {
	OutputText string
}

// ProviderListModelsInput contains provider model-listing input fields.
type ProviderListModelsInput struct {
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// ProviderModel contains one provider model option.
type ProviderModel struct {
	ID      string
	OwnedBy string
	Created int64
}

// Service implements ai.v1 credential and agent services.
//
// It is the orchestration root where credential/grant/agent state is validated,
// authorized, and projected into protocol responses for callers.
type Service struct {
	aiv1.UnimplementedCredentialServiceServer
	aiv1.UnimplementedAgentServiceServer
	aiv1.UnimplementedInvocationServiceServer
	aiv1.UnimplementedCampaignOrchestrationServiceServer
	aiv1.UnimplementedCampaignArtifactServiceServer
	aiv1.UnimplementedSystemReferenceServiceServer
	aiv1.UnimplementedProviderGrantServiceServer
	aiv1.UnimplementedAccessRequestServiceServer

	credentialStore            storage.CredentialStore
	agentStore                 storage.AgentStore
	providerGrantStore         storage.ProviderGrantStore
	connectSessionStore        storage.ProviderConnectSessionStore
	accessRequestStore         storage.AccessRequestStore
	auditEventStore            storage.AuditEventStore
	campaignArtifactManager    *campaigncontext.Manager
	systemReferenceCorpus      *campaigncontext.ReferenceCorpus
	gameCampaignAIClient       gamev1.CampaignAIServiceClient
	gameAuthorizationClient    gamev1.AuthorizationServiceClient
	internalServiceAllowlist   map[string]struct{}
	providerOAuthAdapters      map[providergrant.Provider]ProviderOAuthAdapter
	providerInvocationAdapters map[providergrant.Provider]ProviderInvocationAdapter
	providerToolAdapters       map[providergrant.Provider]orchestration.Provider
	providerModelAdapters      map[providergrant.Provider]ProviderModelAdapter
	campaignTurnRunner         orchestration.CampaignTurnRunner
	sessionGrantConfig         *aisessiongrant.Config
	sealer                     SecretSealer

	clock       func() time.Time
	idGenerator func() (string, error)
	// codeVerifierGenerator is injectable for tests; production uses
	// cryptographic randomness for PKCE verifier generation.
	codeVerifierGenerator func() (string, error)
}

// NewService builds a new ai.v1 service implementation.
//
// Passing the service and credential stores separately allows one persisted
// snapshot to satisfy multiple interfaces while preserving explicit dependency intent.
func NewService(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *Service {
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
	var campaignArtifactStore storage.CampaignArtifactStore
	if store, ok := credentialStore.(storage.CampaignArtifactStore); ok {
		campaignArtifactStore = store
	}
	if campaignArtifactStore == nil {
		if store, ok := agentStore.(storage.CampaignArtifactStore); ok {
			campaignArtifactStore = store
		}
	}
	defaultOpenAIAdapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{})
	providerModelAdapters := map[providergrant.Provider]ProviderModelAdapter{}
	if modelAdapter, ok := defaultOpenAIAdapter.(ProviderModelAdapter); ok {
		providerModelAdapters[providergrant.ProviderOpenAI] = modelAdapter
	}
	providerToolAdapters := map[providergrant.Provider]orchestration.Provider{}
	if toolAdapter, ok := defaultOpenAIAdapter.(orchestration.Provider); ok {
		providerToolAdapters[providergrant.ProviderOpenAI] = toolAdapter
	}
	service := &Service{
		credentialStore:     credentialStore,
		agentStore:          agentStore,
		providerGrantStore:  providerGrantStore,
		connectSessionStore: connectSessionStore,
		accessRequestStore:  accessRequestStore,
		auditEventStore:     auditEventStore,
		providerOAuthAdapters: map[providergrant.Provider]ProviderOAuthAdapter{
			providergrant.ProviderOpenAI: &defaultOpenAIOAuthAdapter{},
		},
		providerInvocationAdapters: map[providergrant.Provider]ProviderInvocationAdapter{
			providergrant.ProviderOpenAI: defaultOpenAIAdapter,
		},
		providerToolAdapters:  providerToolAdapters,
		providerModelAdapters: providerModelAdapters,
		sealer:                sealer,
		clock:                 time.Now,
		idGenerator:           id.NewID,
		codeVerifierGenerator: func() (string, error) {
			return generatePKCECodeVerifier()
		},
	}
	if campaignArtifactStore != nil {
		service.campaignArtifactManager = campaigncontext.NewManager(campaignArtifactStore, time.Now)
	}
	return service
}

// SetGameCampaignAIClient sets the game internal campaign AI client.
func (s *Service) SetGameCampaignAIClient(client gamev1.CampaignAIServiceClient) {
	if s == nil {
		return
	}
	s.gameCampaignAIClient = client
}

// SetGameAuthorizationClient sets the game authorization client used for user-scoped campaign validation.
func (s *Service) SetGameAuthorizationClient(client gamev1.AuthorizationServiceClient) {
	if s == nil {
		return
	}
	s.gameAuthorizationClient = client
}

// SetInternalServiceAllowlist defines which inbound service identities may use
// internal-only campaign-context access without a user-scoped authz hop.
func (s *Service) SetInternalServiceAllowlist(allowlist map[string]struct{}) {
	if s == nil {
		return
	}
	s.internalServiceAllowlist = allowlist
}

// SetCampaignArtifactManager overrides the campaign artifact manager.
func (s *Service) SetCampaignArtifactManager(manager *campaigncontext.Manager) {
	if s == nil || manager == nil {
		return
	}
	s.campaignArtifactManager = manager
}

// SetSystemReferenceCorpus overrides the system reference corpus.
func (s *Service) SetSystemReferenceCorpus(corpus *campaigncontext.ReferenceCorpus) {
	if s == nil || corpus == nil {
		return
	}
	s.systemReferenceCorpus = corpus
}

// SetOpenAIOAuthAdapter overrides the OpenAI OAuth adapter implementation.
func (s *Service) SetOpenAIOAuthAdapter(adapter ProviderOAuthAdapter) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerOAuthAdapters == nil {
		s.providerOAuthAdapters = make(map[providergrant.Provider]ProviderOAuthAdapter)
	}
	s.providerOAuthAdapters[providergrant.ProviderOpenAI] = adapter
}

// SetOpenAIInvocationAdapter overrides the OpenAI invocation adapter.
func (s *Service) SetOpenAIInvocationAdapter(adapter ProviderInvocationAdapter) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerInvocationAdapters == nil {
		s.providerInvocationAdapters = make(map[providergrant.Provider]ProviderInvocationAdapter)
	}
	s.providerInvocationAdapters[providergrant.ProviderOpenAI] = adapter
	if toolAdapter, ok := adapter.(orchestration.Provider); ok {
		if s.providerToolAdapters == nil {
			s.providerToolAdapters = make(map[providergrant.Provider]orchestration.Provider)
		}
		s.providerToolAdapters[providergrant.ProviderOpenAI] = toolAdapter
	}
	if modelAdapter, ok := adapter.(ProviderModelAdapter); ok {
		if s.providerModelAdapters == nil {
			s.providerModelAdapters = make(map[providergrant.Provider]ProviderModelAdapter)
		}
		s.providerModelAdapters[providergrant.ProviderOpenAI] = modelAdapter
	}
}

// SetOpenAICampaignTurnAdapter overrides the OpenAI tool-capable campaign adapter.
func (s *Service) SetOpenAICampaignTurnAdapter(adapter orchestration.Provider) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerToolAdapters == nil {
		s.providerToolAdapters = make(map[providergrant.Provider]orchestration.Provider)
	}
	s.providerToolAdapters[providergrant.ProviderOpenAI] = adapter
}

// SetCampaignTurnRunner overrides the campaign-turn orchestration runner.
func (s *Service) SetCampaignTurnRunner(runner orchestration.CampaignTurnRunner) {
	if s == nil || runner == nil {
		return
	}
	s.campaignTurnRunner = runner
}

// SetAISessionGrantConfig sets the validated session-grant config used by the internal campaign-turn path.
func (s *Service) SetAISessionGrantConfig(cfg aisessiongrant.Config) {
	if s == nil {
		return
	}
	s.sessionGrantConfig = &cfg
}

// SetOpenAIModelAdapter overrides the OpenAI model-listing adapter.
func (s *Service) SetOpenAIModelAdapter(adapter ProviderModelAdapter) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerModelAdapters == nil {
		s.providerModelAdapters = make(map[providergrant.Provider]ProviderModelAdapter)
	}
	s.providerModelAdapters[providergrant.ProviderOpenAI] = adapter
}
