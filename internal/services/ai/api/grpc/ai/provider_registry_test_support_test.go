package ai

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

func mustProviderRegistryForTransportTests(
	t *testing.T,
	oauthAdapters map[provider.Provider]provideroauth.Adapter,
	invocationAdapters map[provider.Provider]provider.InvocationAdapter,
	modelAdapters map[provider.Provider]provider.ModelAdapter,
	toolAdapters map[provider.Provider]orchestration.Provider,
) *providercatalog.Registry {
	t.Helper()

	bundlesByProvider := map[provider.Provider]providercatalog.Bundle{}
	for providerID, adapter := range oauthAdapters {
		bundle := bundlesByProvider[providerID]
		bundle.Provider = providerID
		bundle.OAuth = adapter
		bundlesByProvider[providerID] = bundle
	}
	for providerID, adapter := range invocationAdapters {
		bundle := bundlesByProvider[providerID]
		bundle.Provider = providerID
		bundle.Invocation = adapter
		bundlesByProvider[providerID] = bundle
	}
	for providerID, adapter := range modelAdapters {
		bundle := bundlesByProvider[providerID]
		bundle.Provider = providerID
		bundle.Model = adapter
		bundlesByProvider[providerID] = bundle
	}
	for providerID, adapter := range toolAdapters {
		bundle := bundlesByProvider[providerID]
		bundle.Provider = providerID
		bundle.Tool = adapter
		bundlesByProvider[providerID] = bundle
	}

	bundles := make([]providercatalog.Bundle, 0, len(bundlesByProvider))
	for _, bundle := range bundlesByProvider {
		bundles = append(bundles, bundle)
	}
	registry, err := providercatalog.New(bundles...)
	if err != nil {
		t.Fatalf("providercatalog.New: %v", err)
	}
	return registry
}
