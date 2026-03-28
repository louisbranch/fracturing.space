package ai

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func newCredentialHandlersWithStores(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer) *CredentialHandlers {
	t.Helper()
	return newCredentialHandlersWithOpts(t, credentialStore, agentStore, sealer, nil, nil, nil)
}

func newCredentialHandlersWithOpts(t *testing.T, credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer secret.Sealer, clock service.Clock, idGen service.IDGenerator, usagePolicy *service.UsagePolicy) *CredentialHandlers {
	t.Helper()
	if usagePolicy == nil {
		agentBindingUsageReader := service.NewAgentBindingUsageReader(nil)
		authReferenceUsageReader := service.NewAuthReferenceUsageReader(agentStore, agentBindingUsageReader)
		usagePolicy = service.NewUsagePolicy(service.UsagePolicyConfig{
			AgentBindingUsageReader:  agentBindingUsageReader,
			AuthReferenceUsageReader: authReferenceUsageReader,
		})
	}
	credSvc, err := service.NewCredentialService(service.CredentialServiceConfig{
		CredentialStore:  credentialStore,
		ProviderRegistry: mustProviderRegistryForTransportTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           sealer,
		UsagePolicy:      usagePolicy,
		Clock:            clock,
		IDGenerator:      idGen,
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}
	h, err := NewCredentialHandlers(CredentialHandlersConfig{
		CredentialService: credSvc,
	})
	if err != nil {
		t.Fatalf("NewCredentialHandlers: %v", err)
	}
	return h
}
