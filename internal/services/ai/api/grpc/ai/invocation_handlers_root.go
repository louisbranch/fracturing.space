package ai

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// InvocationHandlers serves invocation RPCs from an explicit invoke-time boundary.
type InvocationHandlers struct {
	aiv1.UnimplementedInvocationServiceServer

	credentialStore            storage.CredentialStore
	agentStore                 storage.AgentStore
	providerGrantStore         storage.ProviderGrantStore
	accessRequestStore         storage.AccessRequestStore
	auditEventStore            storage.AuditEventStore
	providerOAuthAdapters      map[provider.Provider]provider.OAuthAdapter
	providerInvocationAdapters map[provider.Provider]provider.InvocationAdapter
	sealer                     SecretSealer
	clock                      func() time.Time
}

// InvocationHandlersConfig declares the dependencies for invocation RPCs.
type InvocationHandlersConfig struct {
	CredentialStore            storage.CredentialStore
	AgentStore                 storage.AgentStore
	ProviderGrantStore         storage.ProviderGrantStore
	AccessRequestStore         storage.AccessRequestStore
	AuditEventStore            storage.AuditEventStore
	ProviderOAuthAdapters      map[provider.Provider]provider.OAuthAdapter
	ProviderInvocationAdapters map[provider.Provider]provider.InvocationAdapter
	Sealer                     SecretSealer
	Clock                      func() time.Time
}

// NewInvocationHandlers builds an invocation RPC server from explicit deps.
func NewInvocationHandlers(cfg InvocationHandlersConfig) *InvocationHandlers {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	providerOAuthAdapters := newProviderOAuthAdapters(cfg.ProviderOAuthAdapters)
	providerInvocationAdapters := make(map[provider.Provider]provider.InvocationAdapter, len(cfg.ProviderInvocationAdapters))
	for providerID, adapter := range cfg.ProviderInvocationAdapters {
		providerInvocationAdapters[providerID] = adapter
	}

	return &InvocationHandlers{
		credentialStore:            cfg.CredentialStore,
		agentStore:                 cfg.AgentStore,
		providerGrantStore:         cfg.ProviderGrantStore,
		accessRequestStore:         cfg.AccessRequestStore,
		auditEventStore:            cfg.AuditEventStore,
		providerOAuthAdapters:      providerOAuthAdapters,
		providerInvocationAdapters: providerInvocationAdapters,
		sealer:                     cfg.Sealer,
		clock:                      clock,
	}
}
