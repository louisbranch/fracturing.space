package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
)

func TestDefaultOAuthIssuer(t *testing.T) {
	if defaultOAuthIssuer("") != "" {
		t.Fatal("expected empty issuer")
	}
	if defaultOAuthIssuer(":8080") != "http://localhost:8080" {
		t.Fatal("expected localhost prefix for port-only addr")
	}
	if defaultOAuthIssuer("http://example.com/") != "http://example.com" {
		t.Fatal("expected trimmed trailing slash")
	}
	if defaultOAuthIssuer("example.com") != "http://example.com" {
		t.Fatal("expected http prefix for host")
	}
}

func TestOpenAuthStoreInvalidDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	path := filepath.Join(file, "auth.db")

	if _, err := openAuthStore(path); err == nil {
		t.Fatal("expected error for invalid storage dir")
	}
}

func TestBootstrapOAuthUsersSkipsInvalid(t *testing.T) {
	store := openTempAuthStore(t)
	oauthStore := oauth.NewStore(store.DB())
	config := oauth.Config{
		BootstrapUsers: []oauth.BootstrapUser{{Username: "", Password: "", DisplayName: ""}},
	}

	if err := bootstrapOAuthUsers(store, oauthStore, config); err != nil {
		t.Fatalf("bootstrap oauth users: %v", err)
	}
}

func TestBootstrapOAuthUsersNilStores(t *testing.T) {
	config := oauth.Config{
		BootstrapUsers: []oauth.BootstrapUser{{Username: "user", Password: "pass", DisplayName: "User"}},
	}
	if err := bootstrapOAuthUsers(nil, nil, config); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestBootstrapOAuthUsersCreatesCredentials(t *testing.T) {
	store := openTempAuthStore(t)
	oauthStore := oauth.NewStore(store.DB())
	config := oauth.Config{
		BootstrapUsers: []oauth.BootstrapUser{{Username: "user", Password: "pass", DisplayName: "User"}},
	}

	if err := bootstrapOAuthUsers(store, oauthStore, config); err != nil {
		t.Fatalf("bootstrap oauth users: %v", err)
	}

	creds, err := oauthStore.GetOAuthUserByUsername("user")
	if err != nil {
		t.Fatalf("get oauth user: %v", err)
	}
	if creds == nil || creds.UserID == "" {
		t.Fatal("expected credentials to be created")
	}
}

func openTempAuthStore(t *testing.T) *sqlite.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close auth store: %v", err)
		}
	})
	return store
}
