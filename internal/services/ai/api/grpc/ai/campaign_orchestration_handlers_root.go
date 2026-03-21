package ai

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// CampaignOrchestrationHandlers serves campaign-turn orchestration RPCs from an
// explicit runtime boundary instead of the broad transport root.
type CampaignOrchestrationHandlers struct {
	aiv1.UnimplementedCampaignOrchestrationServiceServer

	agentStore              storage.AgentStore
	campaignArtifactManager *campaigncontext.Manager
	gameCampaignAIClient    gamev1.CampaignAIServiceClient
	providerToolAdapters    map[provider.Provider]orchestration.Provider
	campaignTurnRunner      orchestration.CampaignTurnRunner
	sessionGrantConfig      *aisessiongrant.Config
	authTokenResolver       authTokenResolver
}

// CampaignOrchestrationHandlersConfig declares the dependencies for campaign
// orchestration RPCs.
type CampaignOrchestrationHandlersConfig struct {
	AgentStore              storage.AgentStore
	CredentialStore         storage.CredentialStore
	ProviderGrantStore      storage.ProviderGrantStore
	CampaignArtifactManager *campaigncontext.Manager
	GameCampaignAIClient    gamev1.CampaignAIServiceClient
	ProviderOAuthAdapters   map[provider.Provider]provider.OAuthAdapter
	ProviderToolAdapters    map[provider.Provider]orchestration.Provider
	CampaignTurnRunner      orchestration.CampaignTurnRunner
	SessionGrantConfig      *aisessiongrant.Config
	Sealer                  SecretSealer
	Clock                   func() time.Time
}

// NewCampaignOrchestrationHandlers builds a campaign-orchestration RPC server
// from explicit runtime and auth-resolution dependencies.
func NewCampaignOrchestrationHandlers(cfg CampaignOrchestrationHandlersConfig) *CampaignOrchestrationHandlers {
	providerToolAdapters := make(map[provider.Provider]orchestration.Provider, len(cfg.ProviderToolAdapters))
	for providerID, adapter := range cfg.ProviderToolAdapters {
		providerToolAdapters[providerID] = adapter
	}

	var sessionGrantConfig *aisessiongrant.Config
	if cfg.SessionGrantConfig != nil {
		copied := *cfg.SessionGrantConfig
		sessionGrantConfig = &copied
	}

	return &CampaignOrchestrationHandlers{
		agentStore:              cfg.AgentStore,
		campaignArtifactManager: cfg.CampaignArtifactManager,
		gameCampaignAIClient:    cfg.GameCampaignAIClient,
		providerToolAdapters:    providerToolAdapters,
		campaignTurnRunner:      cfg.CampaignTurnRunner,
		sessionGrantConfig:      sessionGrantConfig,
		authTokenResolver: newAuthTokenResolver(
			cfg.CredentialStore,
			cfg.ProviderGrantStore,
			newProviderOAuthAdapters(cfg.ProviderOAuthAdapters),
			cfg.Sealer,
			cfg.Clock,
		),
	}
}
