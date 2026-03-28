package modules

import "testing"

func TestNewDependenciesSetsSharedRuntimeConfig(t *testing.T) {
	t.Parallel()

	deps := NewDependencies("https://cdn.example.com/assets")
	if deps.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("AssetBaseURL = %q, want %q", deps.AssetBaseURL, "https://cdn.example.com/assets")
	}
}
