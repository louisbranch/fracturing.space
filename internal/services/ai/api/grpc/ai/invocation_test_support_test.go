package ai

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

type invocationTestOpts struct {
	clock              service.Clock
	oauthAdapters      map[provider.Provider]provideroauth.Adapter
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
		oauthAdapters = map[provider.Provider]provideroauth.Adapter{
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
	var auditEventStore auditevent.Store
	if store, ok := credentialStore.(auditevent.Store); ok {
		auditEventStore = store
	}
	if auditEventStore == nil {
		if store, ok := agentStore.(auditevent.Store); ok {
			auditEventStore = store
		}
	}

	providerGrantRuntime := service.NewProviderGrantRuntime(service.ProviderGrantRuntimeConfig{
		ProviderGrantStore: providerGrantStore,
		ProviderRegistry:   mustProviderRegistryForTransportTests(t, oauthAdapters, invocationAdapters, nil, nil),
		Sealer:             sealer,
		Clock:              opts.clock,
	})
	authMaterialResolver := service.NewAuthMaterialResolver(service.AuthMaterialResolverConfig{
		CredentialStore:      credentialStore,
		Sealer:               sealer,
		ProviderGrantRuntime: providerGrantRuntime,
	})
	accessibleAgentResolver := service.NewAccessibleAgentResolver(agentStore, accessRequestStore)

	invocationSvc, err := service.NewInvocationService(service.InvocationServiceConfig{
		AgentStore:              agentStore,
		AuditEventStore:         auditEventStore,
		AccessibleAgentResolver: accessibleAgentResolver,
		AuthMaterialResolver:    authMaterialResolver,
		ProviderRegistry:        mustProviderRegistryForTransportTests(t, oauthAdapters, invocationAdapters, nil, nil),
		Clock:                   opts.clock,
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
