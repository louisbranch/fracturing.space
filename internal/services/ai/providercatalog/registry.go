package providercatalog

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

// Bundle declares the runtime capabilities registered for one provider.
type Bundle struct {
	Provider   provider.Provider
	OAuth      provideroauth.Adapter
	Invocation provider.InvocationAdapter
	Model      provider.ModelAdapter
	Tool       orchestration.Provider
}

// Registry holds the runtime provider bundles available to the current
// process.
type Registry struct {
	bundles map[provider.Provider]Bundle
}

// New builds a provider registry from explicit bundles.
func New(bundles ...Bundle) (*Registry, error) {
	registry := &Registry{bundles: make(map[provider.Provider]Bundle, len(bundles))}
	for _, bundle := range bundles {
		normalizedProvider, err := provider.Normalize(string(bundle.Provider))
		if err != nil {
			return nil, fmt.Errorf("normalize provider bundle: %w", err)
		}
		if _, exists := registry.bundles[normalizedProvider]; exists {
			return nil, fmt.Errorf("duplicate provider bundle: %s", normalizedProvider)
		}
		bundle.Provider = normalizedProvider
		registry.bundles[normalizedProvider] = bundle
	}
	return registry, nil
}

// HasProvider reports whether the provider is registered in the runtime.
func (r *Registry) HasProvider(providerID provider.Provider) bool {
	if r == nil {
		return false
	}
	_, ok := r.bundles[providerID]
	return ok
}

// OAuthAdapter returns the registered OAuth adapter for one provider.
func (r *Registry) OAuthAdapter(providerID provider.Provider) (provideroauth.Adapter, bool) {
	if r == nil {
		return nil, false
	}
	bundle, ok := r.bundles[providerID]
	if !ok || bundle.OAuth == nil {
		return nil, false
	}
	return bundle.OAuth, true
}

// InvocationAdapter returns the registered invocation adapter for one provider.
func (r *Registry) InvocationAdapter(providerID provider.Provider) (provider.InvocationAdapter, bool) {
	if r == nil {
		return nil, false
	}
	bundle, ok := r.bundles[providerID]
	if !ok || bundle.Invocation == nil {
		return nil, false
	}
	return bundle.Invocation, true
}

// ModelAdapter returns the registered model adapter for one provider.
func (r *Registry) ModelAdapter(providerID provider.Provider) (provider.ModelAdapter, bool) {
	if r == nil {
		return nil, false
	}
	bundle, ok := r.bundles[providerID]
	if !ok || bundle.Model == nil {
		return nil, false
	}
	return bundle.Model, true
}

// ToolAdapter returns the registered orchestration provider for one provider.
func (r *Registry) ToolAdapter(providerID provider.Provider) (orchestration.Provider, bool) {
	if r == nil {
		return nil, false
	}
	bundle, ok := r.bundles[providerID]
	if !ok || bundle.Tool == nil {
		return nil, false
	}
	return bundle.Tool, true
}
