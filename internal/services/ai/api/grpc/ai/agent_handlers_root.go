package ai

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AgentHandlers serves agent RPCs from an explicit agent-focused boundary.
type AgentHandlers struct {
	aiv1.UnimplementedAgentServiceServer

	credentialStore       storage.CredentialStore
	agentStore            storage.AgentStore
	providerGrantStore    storage.ProviderGrantStore
	accessRequestStore    storage.AccessRequestStore
	providerModelAdapters map[provider.Provider]provider.ModelAdapter
	gameCampaignAIClient  gamev1.CampaignAIServiceClient
	authTokenResolver     authTokenResolver
	clock                 func() time.Time
	idGenerator           func() (string, error)
}

// AgentHandlersConfig declares the dependencies for agent RPCs.
type AgentHandlersConfig struct {
	CredentialStore       storage.CredentialStore
	AgentStore            storage.AgentStore
	ProviderGrantStore    storage.ProviderGrantStore
	AccessRequestStore    storage.AccessRequestStore
	ProviderOAuthAdapters map[provider.Provider]provider.OAuthAdapter
	ProviderModelAdapters map[provider.Provider]provider.ModelAdapter
	GameCampaignAIClient  gamev1.CampaignAIServiceClient
	Sealer                SecretSealer
	Clock                 func() time.Time
	IDGenerator           func() (string, error)
}

// NewAgentHandlers builds an agent RPC server from explicit dependencies.
func NewAgentHandlers(cfg AgentHandlersConfig) *AgentHandlers {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := cfg.IDGenerator
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	providerModelAdapters := make(map[provider.Provider]provider.ModelAdapter, len(cfg.ProviderModelAdapters))
	for providerID, adapter := range cfg.ProviderModelAdapters {
		providerModelAdapters[providerID] = adapter
	}

	return &AgentHandlers{
		credentialStore:       cfg.CredentialStore,
		agentStore:            cfg.AgentStore,
		providerGrantStore:    cfg.ProviderGrantStore,
		accessRequestStore:    cfg.AccessRequestStore,
		providerModelAdapters: providerModelAdapters,
		gameCampaignAIClient:  cfg.GameCampaignAIClient,
		authTokenResolver: newAuthTokenResolver(
			cfg.CredentialStore,
			cfg.ProviderGrantStore,
			newProviderOAuthAdapters(cfg.ProviderOAuthAdapters),
			cfg.Sealer,
			clock,
		),
		clock:       clock,
		idGenerator: idGenerator,
	}
}
