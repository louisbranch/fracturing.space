package cache

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	websqlite "github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite"
)

// OpenStore opens the web cache store when a storage path is provided.
func OpenStore(path string) (*websqlite.Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create web cache dir: %w", err)
		}
	}
	store, err := websqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open web cache sqlite store: %w", err)
	}
	return store, nil
}

// BuildAuthConsentURL resolves the post-magic-link consent callback URL.
func BuildAuthConsentURL(base string, pendingID string) string {
	base = strings.TrimSpace(base)
	encoded := url.QueryEscape(pendingID)
	if base == "" {
		return "/authorize/consent?pending_id=" + encoded
	}
	return strings.TrimRight(base, "/") + "/authorize/consent?pending_id=" + encoded
}
