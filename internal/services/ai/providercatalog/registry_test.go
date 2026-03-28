package providercatalog

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

func TestRegistryRejectsDuplicateProviderBundles(t *testing.T) {
	_, err := New(
		Bundle{Provider: provider.OpenAI},
		Bundle{Provider: provider.OpenAI},
	)
	if err == nil {
		t.Fatal("expected duplicate bundle error")
	}
}

func TestRegistryHasProvider(t *testing.T) {
	registry, err := New(Bundle{Provider: provider.OpenAI})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !registry.HasProvider(provider.OpenAI) {
		t.Fatal("expected openai provider to be registered")
	}
	if registry.HasProvider(provider.Anthropic) {
		t.Fatal("did not expect anthropic provider to be registered")
	}
}
