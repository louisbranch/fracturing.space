package ai

import (
	"context"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
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

// ProviderInvokeInput contains provider invocation input fields.
type ProviderInvokeInput struct {
	Model string
	Input string
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// ProviderInvokeResult contains invocation output.
type ProviderInvokeResult struct {
	OutputText string
}

// Service implements ai.v1 credential and agent services.
//
// It is the orchestration root where credential/grant/agent state is validated,
// authorized, and projected into protocol responses for callers.
type Service struct {
	aiv1.UnimplementedCredentialServiceServer
	aiv1.UnimplementedAgentServiceServer
	aiv1.UnimplementedInvocationServiceServer
	aiv1.UnimplementedProviderGrantServiceServer
	aiv1.UnimplementedAccessRequestServiceServer

	credentialStore            storage.CredentialStore
	agentStore                 storage.AgentStore
	providerGrantStore         storage.ProviderGrantStore
	connectSessionStore        storage.ProviderConnectSessionStore
	accessRequestStore         storage.AccessRequestStore
	auditEventStore            storage.AuditEventStore
	providerOAuthAdapters      map[providergrant.Provider]ProviderOAuthAdapter
	providerInvocationAdapters map[providergrant.Provider]ProviderInvocationAdapter
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
	return &Service{
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
			providergrant.ProviderOpenAI: NewOpenAIInvokeAdapter(OpenAIInvokeConfig{}),
		},
		sealer:      sealer,
		clock:       time.Now,
		idGenerator: id.NewID,
		codeVerifierGenerator: func() (string, error) {
			return generatePKCECodeVerifier()
		},
	}
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
}
