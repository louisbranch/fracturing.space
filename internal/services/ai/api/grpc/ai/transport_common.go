package ai

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
)

const (
	// userIDHeader is injected by trusted edge/auth layers and consumed here for
	// ownership enforcement. Direct callers must not be allowed to spoof it.
	userIDHeader = "x-fracturing-space-user-id"

	defaultPageSize = 10
	maxPageSize     = 50

	providerGrantRefreshWindow = 2 * time.Minute
)

// SecretSealer is the transport-level alias for secret.Sealer. The canonical
// interface lives in the secret package; handler code uses this alias so
// callers and tests continue compiling without an import change.
type SecretSealer = secret.Sealer

func newProviderOAuthAdapters(adapters map[provider.Provider]provider.OAuthAdapter) map[provider.Provider]provider.OAuthAdapter {
	normalized := make(map[provider.Provider]provider.OAuthAdapter, len(adapters))
	for providerID, adapter := range adapters {
		normalized[providerID] = adapter
	}
	return normalized
}
