package ai

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

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

type campaignOrchestrationTestOpts struct {
	clock                   service.Clock
	oauthAdapters           map[provider.Provider]provideroauth.Adapter
	toolAdapters            map[provider.Provider]orchestration.Provider
	campaignTurnRunner      orchestration.CampaignTurnRunner
	sessionGrantConfig      *aisessiongrant.Config
	campaignAuthStateReader service.CampaignAuthStateReader
}

func newCampaignOrchestrationHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts campaignOrchestrationTestOpts) *CampaignOrchestrationHandlers {
	t.Helper()

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provideroauth.Adapter{
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

	providerGrantRuntime := service.NewProviderGrantRuntime(service.ProviderGrantRuntimeConfig{
		ProviderGrantStore: providerGrantStore,
		ProviderRegistry:   mustProviderRegistryForTransportTests(t, oauthAdapters, nil, nil, toolAdapters),
		Sealer:             sealer,
		Clock:              opts.clock,
	})
	authMaterialResolver := service.NewAuthMaterialResolver(service.AuthMaterialResolverConfig{
		CredentialStore:      credentialStore,
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})

	orchestrationSvc, err := service.NewCampaignOrchestrationService(service.CampaignOrchestrationServiceConfig{
		AgentStore:              agentStore,
		CampaignAuthStateReader: opts.campaignAuthStateReader,
		ProviderRegistry:        mustProviderRegistryForTransportTests(t, oauthAdapters, nil, nil, toolAdapters),
		CampaignTurnRunner:      opts.campaignTurnRunner,
		SessionGrantConfig:      opts.sessionGrantConfig,
		AuthMaterialResolver:    authMaterialResolver,
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
