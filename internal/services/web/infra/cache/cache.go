package cache

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/integration/cache"
	websqlite "github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite"
)

// OpenStore opens the web cache store when a path is configured.
func OpenStore(path string) (*websqlite.Store, error) {
	return cache.OpenStore(path)
}

// BuildAuthConsentURL resolves the post-magic-link consent callback URL.
func BuildAuthConsentURL(base string, pendingID string) string {
	return cache.BuildAuthConsentURL(base, pendingID)
}
