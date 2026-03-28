package ai

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

type agentTestOpts struct {
	clock               service.Clock
	idGenerator         service.IDGenerator
	oauthAdapters       map[provider.Provider]provideroauth.Adapter
	modelAdapters       map[provider.Provider]provider.ModelAdapter
	campaignUsageReader service.CampaignUsageReader
}

func newAgentHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *AgentHandlers {
	t.Helper()
	return newAgentHandlersWithOpts(t, credentialStore, agentStore, sealer, agentTestOpts{})
}

func newAgentHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts agentTestOpts) *AgentHandlers {
	t.Helper()

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provideroauth.Adapter{
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

	providerGrantRuntime := service.NewProviderGrantRuntime(service.ProviderGrantRuntimeConfig{
		ProviderGrantStore: providerGrantStore,
		ProviderRegistry:   mustProviderRegistryForTransportTests(t, oauthAdapters, nil, modelAdapters, nil),
		Sealer:             sealer,
		Clock:              opts.clock,
	})
	authMaterialResolver := service.NewAuthMaterialResolver(service.AuthMaterialResolverConfig{
		CredentialStore:      credentialStore,
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})
	authReferencePolicy, err := service.NewAuthReferencePolicy(service.AuthReferencePolicyConfig{
		CredentialStore:      credentialStore,
		ProviderGrantStore:   providerGrantStore,
		ProviderRegistry:     mustProviderRegistryForTransportTests(t, oauthAdapters, nil, modelAdapters, nil),
		AuthMaterialResolver: authMaterialResolver,
	})
	if err != nil {
		t.Fatalf("NewAuthReferencePolicy: %v", err)
	}
	accessibleAgentResolver := service.NewAccessibleAgentResolver(agentStore, accessRequestStore)

	agentBindingUsageReader := service.NewAgentBindingUsageReader(opts.campaignUsageReader)
	authReferenceUsageReader := service.NewAuthReferenceUsageReader(agentStore, agentBindingUsageReader)
	usagePolicy := service.NewUsagePolicy(service.UsagePolicyConfig{
		AgentBindingUsageReader:  agentBindingUsageReader,
		AuthReferenceUsageReader: authReferenceUsageReader,
	})

	agentSvc, err := service.NewAgentService(service.AgentServiceConfig{
		AgentStore:              agentStore,
		AuthReferencePolicy:     authReferencePolicy,
		AccessibleAgentResolver: accessibleAgentResolver,
		UsagePolicy:             usagePolicy,
		AgentBindingUsageReader: agentBindingUsageReader,
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
