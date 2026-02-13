//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
)

func TestOAuthStaticAssets(t *testing.T) {
	store := openAuthStore(t)
	defer store.Close()

	oauthStore := oauth.NewStore(store.DB())
	server := oauth.NewServer(oauth.Config{}, oauthStore, nil)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, httpServer.URL+"/static/theme.css", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get static theme css: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/css") {
		t.Fatalf("expected text/css content-type, got %q", contentType)
	}
}

func openAuthStore(t *testing.T) *authsqlite.Store {
	t.Helper()

	base := t.TempDir()
	path := filepath.Join(base, "auth.db")
	store, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth sqlite store: %v", err)
	}
	return store
}
