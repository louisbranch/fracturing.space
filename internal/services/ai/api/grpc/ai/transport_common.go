package ai

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

const (
	// userIDHeader is injected by trusted edge/auth layers and consumed here for
	// ownership enforcement. Direct callers must not be allowed to spoof it.
	userIDHeader = "x-fracturing-space-user-id"

	defaultPageSize = 10
	maxPageSize     = 50

	providerGrantRefreshWindow = 2 * time.Minute
)

// SecretSealer encrypts secret values before persistence.
type SecretSealer interface {
	Seal(value string) (string, error)
	Open(sealed string) (string, error)
}

func newProviderOAuthAdapters(adapters map[provider.Provider]provider.OAuthAdapter) map[provider.Provider]provider.OAuthAdapter {
	normalized := make(map[provider.Provider]provider.OAuthAdapter, len(adapters))
	for providerID, adapter := range adapters {
		normalized[providerID] = adapter
	}
	return normalized
}
