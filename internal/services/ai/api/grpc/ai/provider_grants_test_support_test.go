package ai

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// providerGrantTestHandlers bundles the service and handler so tests can
// configure clock/ID generator on the service while calling RPC methods on
// the handler.
type providerGrantTestHandlers struct {
	*ProviderGrantHandlers
	svc *service.ProviderGrantService
}

type providerGrantTestOpts struct {
	clock                 service.Clock
	idGenerator           service.IDGenerator
	codeVerifierGenerator service.CodeVerifierGenerator
	oauthAdapters         map[provider.Provider]provideroauth.Adapter
	usagePolicy           *service.UsagePolicy
}

func newProviderGrantHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *providerGrantTestHandlers {
	t.Helper()
	return newProviderGrantHandlersWithOpts(t, credentialStore, agentStore, sealer, providerGrantTestOpts{})
}

func newProviderGrantHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, opts providerGrantTestOpts) *providerGrantTestHandlers {
	t.Helper()

	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}
	var connectSessionStore providerconnect.Store
	if store, ok := credentialStore.(providerconnect.Store); ok {
		connectSessionStore = store
	}
	if connectSessionStore == nil {
		if store, ok := agentStore.(providerconnect.Store); ok {
			connectSessionStore = store
		}
	}
	var connectFinisher service.ProviderConnectFinisher
	if store, ok := credentialStore.(service.ProviderConnectFinisher); ok {
		connectFinisher = store
	}
	if connectFinisher == nil {
		if store, ok := agentStore.(service.ProviderConnectFinisher); ok {
			connectFinisher = store
		}
	}
	if connectFinisher == nil && providerGrantStore != nil && connectSessionStore != nil {
		connectFinisher = providerGrantTestConnectFinisher{
			providerGrantStore:  providerGrantStore,
			connectSessionStore: connectSessionStore,
		}
	}

	oauthAdapters := opts.oauthAdapters
	if oauthAdapters == nil {
		oauthAdapters = map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: &defaultProviderOAuthAdapterForTests{},
		}
	}
	usagePolicy := opts.usagePolicy
	if usagePolicy == nil {
		agentBindingUsageReader := service.NewAgentBindingUsageReader(nil)
		authReferenceUsageReader := service.NewAuthReferenceUsageReader(agentStore, agentBindingUsageReader)
		usagePolicy = service.NewUsagePolicy(service.UsagePolicyConfig{
			AgentBindingUsageReader:  agentBindingUsageReader,
			AuthReferenceUsageReader: authReferenceUsageReader,
		})
	}

	svc, err := service.NewProviderGrantService(service.ProviderGrantServiceConfig{
		ProviderGrantStore:    providerGrantStore,
		ConnectSessionStore:   connectSessionStore,
		ConnectFinisher:       connectFinisher,
		Sealer:                sealer,
		ProviderRegistry:      mustProviderRegistryForTransportTests(t, oauthAdapters, nil, nil, nil),
		UsagePolicy:           usagePolicy,
		Clock:                 opts.clock,
		IDGenerator:           opts.idGenerator,
		CodeVerifierGenerator: opts.codeVerifierGenerator,
	})
	if err != nil {
		t.Fatalf("NewProviderGrantService: %v", err)
	}
	h, err := NewProviderGrantHandlers(ProviderGrantHandlersConfig{
		ProviderGrantService: svc,
	})
	if err != nil {
		t.Fatalf("NewProviderGrantHandlers: %v", err)
	}
	return &providerGrantTestHandlers{ProviderGrantHandlers: h, svc: svc}
}

type providerGrantTestConnectFinisher struct {
	providerGrantStore  storage.ProviderGrantStore
	connectSessionStore providerconnect.Store
}

func (f providerGrantTestConnectFinisher) FinishProviderConnect(ctx context.Context, grant providergrant.ProviderGrant, completedSession providerconnect.Session) error {
	session, err := f.connectSessionStore.GetProviderConnectSession(ctx, completedSession.ID)
	if err != nil {
		return err
	}
	if session.OwnerUserID != completedSession.OwnerUserID || session.Status != providerconnect.StatusPending || completedSession.Status != providerconnect.StatusCompleted {
		return storage.ErrNotFound
	}
	if err := f.providerGrantStore.PutProviderGrant(ctx, grant); err != nil {
		return err
	}
	return f.connectSessionStore.CompleteProviderConnectSession(ctx, completedSession)
}
