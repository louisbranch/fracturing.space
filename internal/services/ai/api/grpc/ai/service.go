package ai

import (
	"context"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
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
	Model           string
	Input           string
	Instructions    string
	ReasoningEffort string
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// ProviderInvokeResult contains invocation output.
type ProviderInvokeResult struct {
	OutputText string
	Usage      provider.Usage
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

func newProviderOAuthAdapters(adapters map[provider.Provider]ProviderOAuthAdapter) map[provider.Provider]ProviderOAuthAdapter {
	normalized := make(map[provider.Provider]ProviderOAuthAdapter, len(adapters))
	for providerID, adapter := range adapters {
		normalized[providerID] = adapter
	}
	return normalized
}

// ServiceConfig declares the explicit dependencies for the AI transport root.
type ServiceConfig struct {
	CredentialStore         storage.CredentialStore
	AgentStore              storage.AgentStore
	ProviderGrantStore      storage.ProviderGrantStore
	ConnectSessionStore     storage.ProviderConnectSessionStore
	AccessRequestStore      storage.AccessRequestStore
	AuditEventStore         storage.AuditEventStore
	CampaignArtifactStore   storage.CampaignArtifactStore
	CampaignArtifactManager *campaigncontext.Manager
	Sealer                  SecretSealer

	ProviderOAuthAdapters      map[provider.Provider]ProviderOAuthAdapter
	ProviderInvocationAdapters map[provider.Provider]ProviderInvocationAdapter
	ProviderToolAdapters       map[provider.Provider]orchestration.Provider
	ProviderModelAdapters      map[provider.Provider]ProviderModelAdapter

	Clock                 func() time.Time
	IDGenerator           func() (string, error)
	CodeVerifierGenerator func() (string, error)
}

// Service implements ai.v1 credential and agent services.
//
// It is the orchestration root where credential/grant/agent state is validated,
// authorized, and projected into protocol responses for callers.
type Service struct {
	aiv1.UnimplementedAgentServiceServer
	aiv1.UnimplementedInvocationServiceServer
	aiv1.UnimplementedCampaignOrchestrationServiceServer
	aiv1.UnimplementedAccessRequestServiceServer

	credentialStore            storage.CredentialStore
	agentStore                 storage.AgentStore
	providerGrantStore         storage.ProviderGrantStore
	connectSessionStore        storage.ProviderConnectSessionStore
	accessRequestStore         storage.AccessRequestStore
	auditEventStore            storage.AuditEventStore
	campaignArtifactManager    *campaigncontext.Manager
	gameCampaignAIClient       gamev1.CampaignAIServiceClient
	providerOAuthAdapters      map[provider.Provider]ProviderOAuthAdapter
	providerInvocationAdapters map[provider.Provider]ProviderInvocationAdapter
	providerToolAdapters       map[provider.Provider]orchestration.Provider
	providerModelAdapters      map[provider.Provider]ProviderModelAdapter
	campaignTurnRunner         orchestration.CampaignTurnRunner
	sessionGrantConfig         *aisessiongrant.Config
	sealer                     SecretSealer

	clock       func() time.Time
	idGenerator func() (string, error)
	// codeVerifierGenerator is injectable for tests; production uses
	// cryptographic randomness for PKCE verifier generation.
	codeVerifierGenerator func() (string, error)
}

// NewService builds a new ai.v1 service implementation from explicit deps.
func NewService(cfg ServiceConfig) *Service {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := cfg.IDGenerator
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	codeVerifierGenerator := cfg.CodeVerifierGenerator
	if codeVerifierGenerator == nil {
		codeVerifierGenerator = generatePKCECodeVerifier
	}

	providerOAuthAdapters := newProviderOAuthAdapters(cfg.ProviderOAuthAdapters)

	providerInvocationAdapters := make(map[provider.Provider]ProviderInvocationAdapter, len(cfg.ProviderInvocationAdapters))
	for providerID, adapter := range cfg.ProviderInvocationAdapters {
		providerInvocationAdapters[providerID] = adapter
	}

	providerToolAdapters := make(map[provider.Provider]orchestration.Provider, len(cfg.ProviderToolAdapters))
	for providerID, adapter := range cfg.ProviderToolAdapters {
		providerToolAdapters[providerID] = adapter
	}

	providerModelAdapters := make(map[provider.Provider]ProviderModelAdapter, len(cfg.ProviderModelAdapters))
	for providerID, adapter := range cfg.ProviderModelAdapters {
		providerModelAdapters[providerID] = adapter
	}

	service := &Service{
		credentialStore:            cfg.CredentialStore,
		agentStore:                 cfg.AgentStore,
		providerGrantStore:         cfg.ProviderGrantStore,
		connectSessionStore:        cfg.ConnectSessionStore,
		accessRequestStore:         cfg.AccessRequestStore,
		auditEventStore:            cfg.AuditEventStore,
		providerOAuthAdapters:      providerOAuthAdapters,
		providerInvocationAdapters: providerInvocationAdapters,
		providerToolAdapters:       providerToolAdapters,
		providerModelAdapters:      providerModelAdapters,
		sealer:                     cfg.Sealer,
		clock:                      clock,
		idGenerator:                idGenerator,
		codeVerifierGenerator:      codeVerifierGenerator,
	}
	if cfg.CampaignArtifactManager != nil {
		service.campaignArtifactManager = cfg.CampaignArtifactManager
	} else if cfg.CampaignArtifactStore != nil {
		service.campaignArtifactManager = campaigncontext.NewManager(cfg.CampaignArtifactStore, clock)
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
