package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
)

func TestIsAllowedRedirect(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		allowlist []string
		want      bool
	}{
		{"empty allowlist", "http://a.com/cb", nil, false},
		{"match", "http://a.com/cb", []string{"http://a.com/cb"}, true},
		{"no match", "http://b.com/cb", []string{"http://a.com/cb"}, false},
		{"trimmed match", "http://a.com/cb", []string{" http://a.com/cb "}, true},
		{"multiple entries", "http://b.com/cb", []string{"http://a.com/cb", "http://b.com/cb"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowedRedirect(tc.uri, tc.allowlist); got != tc.want {
				t.Errorf("isAllowedRedirect(%q, %v) = %v, want %v", tc.uri, tc.allowlist, got, tc.want)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"first non-empty", []string{"a", "b"}, "a"},
		{"skip empty", []string{"", "b"}, "b"},
		{"skip whitespace", []string{"  ", "c"}, "c"},
		{"all empty", []string{"", "  "}, "Unknown User"},
		{"no values", nil, "Unknown User"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := firstNonEmpty(tc.values...); got != tc.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tc.values, got, tc.want)
			}
		})
	}
}

func TestDerivePrimaryEmail(t *testing.T) {
	t.Run("uses provided email", func(t *testing.T) {
		profile := providerProfile{
			Email:          "Alice@Example.Com ",
			DisplayName:    "Test User",
			ProviderUserID: "provider-1",
		}
		got := derivePrimaryEmail(profile)
		if got != "alice@example.com" {
			t.Fatalf("derivePrimaryEmail = %q, want alice@example.com", got)
		}
	})

	t.Run("falls back to display name and appends provider id", func(t *testing.T) {
		profile := providerProfile{
			DisplayName:    "Ada Lovelace",
			ProviderUserID: "provider-123",
		}
		got := derivePrimaryEmail(profile)
		if got != "ada-lovelace-provider-123@oauth.local" {
			t.Fatalf("derivePrimaryEmail = %q, want %q", got, "ada-lovelace-provider-123@oauth.local")
		}
	})

	t.Run("uses provider id when display name empty", func(t *testing.T) {
		profile := providerProfile{
			ProviderUserID: "provider-123",
		}
		got := derivePrimaryEmail(profile)
		if got != "provider-123@oauth.local" {
			t.Fatalf("derivePrimaryEmail = %q, want %q", got, "provider-123@oauth.local")
		}
	})

	t.Run("adds stable uniqueness for provider users with shared display names", func(t *testing.T) {
		first := derivePrimaryEmail(providerProfile{
			DisplayName:    "Tester",
			ProviderUserID: "provider-11",
		})
		second := derivePrimaryEmail(providerProfile{
			DisplayName:    "Tester",
			ProviderUserID: "provider-22",
		})
		if strings.TrimSpace(first) == "" || strings.TrimSpace(second) == "" {
			t.Fatal("expected non-empty email values")
		}
		if first == second {
			t.Fatalf("expected unique fallback emails, got equal %q", first)
		}
		if first == "tester@oauth.local" || second == "tester@oauth.local" {
			t.Fatal("expected uniqueness suffix when provider id is available")
		}
	})
}

func TestFormatGitHubID(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		want  string
	}{
		{"zero", 0, ""},
		{"positive", 12345, "github-12345"},
		{"negative", -1, "github--1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatGitHubID(tc.value); got != tc.want {
				t.Errorf("formatGitHubID(%d) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestNewCodeVerifier(t *testing.T) {
	v, err := newCodeVerifier()
	if err != nil {
		t.Fatalf("newCodeVerifier: %v", err)
	}
	if len(v) == 0 {
		t.Fatal("expected non-empty verifier")
	}
	// Hex-encoded 48 bytes = 96 characters.
	if len(v) != 96 {
		t.Errorf("expected 96-char verifier, got %d", len(v))
	}

	// Uniqueness check.
	v2, err := newCodeVerifier()
	if err != nil {
		t.Fatalf("newCodeVerifier second call: %v", err)
	}
	if v == v2 {
		t.Error("expected unique verifiers")
	}
}

func TestExchangeProviderToken(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success with expires_in", func(t *testing.T) {
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected form content type, got %q", ct)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.FormValue("grant_type") != "authorization_code" {
				t.Errorf("wrong grant_type: %q", r.FormValue("grant_type"))
			}
			if r.FormValue("code") != "test-code" {
				t.Errorf("wrong code: %q", r.FormValue("code"))
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "tok-123",
				"refresh_token": "ref-456",
				"scope":         "openid",
				"expires_in":    3600,
				"id_token":      "id-789",
			})
		}))
		defer tokenServer.Close()

		s := &Server{
			clock:      func() time.Time { return fixedTime },
			httpClient: tokenServer.Client(),
		}
		provider := ProviderConfig{
			TokenURL:     tokenServer.URL,
			ClientID:     "cid",
			ClientSecret: "csec",
			RedirectURI:  "http://localhost/cb",
		}

		tok, err := s.exchangeProviderToken(context.Background(), provider, "test-code", "verifier")
		if err != nil {
			t.Fatalf("exchangeProviderToken: %v", err)
		}
		if tok.AccessToken != "tok-123" {
			t.Errorf("access_token = %q, want tok-123", tok.AccessToken)
		}
		if tok.RefreshToken != "ref-456" {
			t.Errorf("refresh_token = %q, want ref-456", tok.RefreshToken)
		}
		if tok.IDToken != "id-789" {
			t.Errorf("id_token = %q, want id-789", tok.IDToken)
		}
		expectedExpiry := fixedTime.Add(3600 * time.Second)
		if !tok.ExpiresAt.Equal(expectedExpiry) {
			t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, expectedExpiry)
		}
	})

	t.Run("success without expires_in", func(t *testing.T) {
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-abc",
			})
		}))
		defer tokenServer.Close()

		s := &Server{
			clock:      func() time.Time { return fixedTime },
			httpClient: tokenServer.Client(),
		}
		provider := ProviderConfig{TokenURL: tokenServer.URL}

		tok, err := s.exchangeProviderToken(context.Background(), provider, "code", "verifier")
		if err != nil {
			t.Fatalf("exchangeProviderToken: %v", err)
		}
		if !tok.ExpiresAt.IsZero() {
			t.Errorf("expected zero ExpiresAt, got %v", tok.ExpiresAt)
		}
	})

	t.Run("non-200 response", func(t *testing.T) {
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer tokenServer.Close()

		s := &Server{
			clock:      func() time.Time { return fixedTime },
			httpClient: tokenServer.Client(),
		}
		provider := ProviderConfig{TokenURL: tokenServer.URL}

		_, err := s.exchangeProviderToken(context.Background(), provider, "code", "verifier")
		if err == nil {
			t.Fatal("expected error for non-200 response")
		}
	})

	t.Run("missing access token in response", func(t *testing.T) {
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"scope": "openid"})
		}))
		defer tokenServer.Close()

		s := &Server{
			clock:      func() time.Time { return fixedTime },
			httpClient: tokenServer.Client(),
		}
		provider := ProviderConfig{TokenURL: tokenServer.URL}

		_, err := s.exchangeProviderToken(context.Background(), provider, "code", "verifier")
		if err == nil {
			t.Fatal("expected error for missing access token")
		}
	})
}

func TestFetchProviderProfile(t *testing.T) {
	t.Run("Google provider", func(t *testing.T) {
		profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth := r.Header.Get("Authorization"); auth != "Bearer google-tok" {
				t.Errorf("wrong auth header: %q", auth)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"sub":   "goog-123",
				"name":  "Alice",
				"email": "alice@example.com",
			})
		}))
		defer profileServer.Close()

		s := &Server{httpClient: profileServer.Client()}
		provider := ProviderConfig{Name: "Google", UserInfoURL: profileServer.URL}

		profile, err := s.fetchProviderProfile(context.Background(), provider, "google-tok")
		if err != nil {
			t.Fatalf("fetchProviderProfile: %v", err)
		}
		if profile.ProviderUserID != "goog-123" {
			t.Errorf("ProviderUserID = %q, want goog-123", profile.ProviderUserID)
		}
		if profile.DisplayName != "Alice" {
			t.Errorf("DisplayName = %q, want Alice", profile.DisplayName)
		}
	})

	t.Run("Google provider fallback to email", func(t *testing.T) {
		profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"sub":   "goog-456",
				"email": "bob@example.com",
			})
		}))
		defer profileServer.Close()

		s := &Server{httpClient: profileServer.Client()}
		provider := ProviderConfig{Name: "google", UserInfoURL: profileServer.URL}

		profile, err := s.fetchProviderProfile(context.Background(), provider, "tok")
		if err != nil {
			t.Fatalf("fetchProviderProfile: %v", err)
		}
		if profile.DisplayName != "bob@example.com" {
			t.Errorf("DisplayName = %q, want bob@example.com", profile.DisplayName)
		}
	})

	t.Run("GitHub provider", func(t *testing.T) {
		profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":    42,
				"login": "octocat",
				"name":  "The Octocat",
				"email": "octo@github.com",
			})
		}))
		defer profileServer.Close()

		s := &Server{httpClient: profileServer.Client()}
		provider := ProviderConfig{Name: "GitHub", UserInfoURL: profileServer.URL}

		profile, err := s.fetchProviderProfile(context.Background(), provider, "gh-tok")
		if err != nil {
			t.Fatalf("fetchProviderProfile: %v", err)
		}
		if profile.ProviderUserID != "github-42" {
			t.Errorf("ProviderUserID = %q, want github-42", profile.ProviderUserID)
		}
		if profile.DisplayName != "The Octocat" {
			t.Errorf("DisplayName = %q, want The Octocat", profile.DisplayName)
		}
	})

	t.Run("non-200 response", func(t *testing.T) {
		profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer profileServer.Close()

		s := &Server{httpClient: profileServer.Client()}
		provider := ProviderConfig{Name: "GitHub", UserInfoURL: profileServer.URL}

		_, err := s.fetchProviderProfile(context.Background(), provider, "bad-tok")
		if err == nil {
			t.Fatal("expected error for non-200 response")
		}
	})
}

func TestEnsureUserForProfile(t *testing.T) {
	t.Run("empty provider user ID", func(t *testing.T) {
		s := &Server{}
		_, err := s.ensureUserForProfile(context.Background(), "github", providerProfile{})
		if err == nil {
			t.Fatal("expected error for empty provider user id")
		}
	})

	t.Run("existing identity returns user ID", func(t *testing.T) {
		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })
		oauthStore := NewStore(authStore.DB())

		fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		s := &Server{
			store:     oauthStore,
			userStore: authStore,
			clock:     func() time.Time { return fixedTime },
		}

		// Seed an external identity.
		err = oauthStore.UpsertExternalIdentity(ExternalIdentity{
			ID:             "eid-1",
			Provider:       "github",
			ProviderUserID: "github-42",
			UserID:         "user-existing",
			AccessToken:    "tok",
			ExpiresAt:      fixedTime.Add(time.Hour),
			UpdatedAt:      fixedTime,
		})
		if err != nil {
			t.Fatalf("upsert identity: %v", err)
		}

		userID, err := s.ensureUserForProfile(context.Background(), "github", providerProfile{
			ProviderUserID: "github-42",
			DisplayName:    "Octocat",
		})
		if err != nil {
			t.Fatalf("ensureUserForProfile: %v", err)
		}
		if userID != "user-existing" {
			t.Errorf("userID = %q, want user-existing", userID)
		}
	})

	t.Run("new identity creates user", func(t *testing.T) {
		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })
		oauthStore := NewStore(authStore.DB())

		fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		s := &Server{
			store:     oauthStore,
			userStore: authStore,
			clock:     func() time.Time { return fixedTime },
		}

		userID, err := s.ensureUserForProfile(context.Background(), "github", providerProfile{
			ProviderUserID: "github-99",
			DisplayName:    "New User",
		})
		if err != nil {
			t.Fatalf("ensureUserForProfile: %v", err)
		}
		if userID == "" {
			t.Fatal("expected non-empty user ID")
		}
	})

	t.Run("nil user store", func(t *testing.T) {
		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })
		oauthStore := NewStore(authStore.DB())

		s := &Server{
			store:     oauthStore,
			userStore: nil,
			clock:     func() time.Time { return time.Now() },
		}

		_, err = s.ensureUserForProfile(context.Background(), "github", providerProfile{
			ProviderUserID: "github-99",
			DisplayName:    "New User",
		})
		if err == nil {
			t.Fatal("expected error when user store is nil")
		}
	})
}

func TestHandleProviderRoutes(t *testing.T) {
	t.Run("too few path parts", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github", nil)
		w := httptest.NewRecorder()
		server.handleProviderRoutes(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/unknown", nil)
		w := httptest.NewRecorder()
		server.handleProviderRoutes(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("routes to start", func(t *testing.T) {
		server, _ := testServer(t)
		// No provider configured, so should return 404.
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/start", nil)
		w := httptest.NewRecorder()
		server.handleProviderRoutes(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for unconfigured provider, got %d", w.Code)
		}
	})

	t.Run("routes to callback", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback", nil)
		w := httptest.NewRecorder()
		server.handleProviderRoutes(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for unconfigured provider, got %d", w.Code)
		}
	})
}

func TestHandleProviderStart(t *testing.T) {
	providerConfig := func() (Config, *httptest.Server) {
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		cfg := testServerConfig()
		cfg.Providers = map[string]ProviderConfig{
			"github": {
				Name:         "GitHub",
				ClientID:     "gh-client",
				ClientSecret: "gh-secret",
				RedirectURI:  "http://localhost/cb",
				AuthURL:      authServer.URL,
				TokenURL:     authServer.URL + "/token",
				UserInfoURL:  authServer.URL + "/user",
				Scopes:       []string{"read:user"},
			},
		}
		cfg.PendingAuthorizationTTL = 10 * time.Minute
		return cfg, authServer
	}

	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/oauth/providers/github/start", nil)
		w := httptest.NewRecorder()
		server.handleProviderStart(w, req, "github")
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/unknown/start", nil)
		w := httptest.NewRecorder()
		server.handleProviderStart(w, req, "unknown")
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("disallowed redirect_uri", func(t *testing.T) {
		cfg, authServer := providerConfig()
		defer authServer.Close()
		cfg.LoginRedirectAllowlist = []string{"http://allowed.com/cb"}

		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		server := NewServer(cfg, NewStore(authStore.DB()), authStore)

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/start?redirect_uri=http://evil.com/cb", nil)
		w := httptest.NewRecorder()
		server.handleProviderStart(w, req, "github")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("success redirects to provider", func(t *testing.T) {
		cfg, authServer := providerConfig()
		defer authServer.Close()

		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		server := NewServer(cfg, NewStore(authStore.DB()), authStore)

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/start", nil)
		w := httptest.NewRecorder()
		server.handleProviderStart(w, req, "github")
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if !strings.HasPrefix(location, authServer.URL) {
			t.Errorf("expected redirect to auth server, got %q", location)
		}
		if !strings.Contains(location, "client_id=gh-client") {
			t.Errorf("expected client_id in redirect, got %q", location)
		}
		if !strings.Contains(location, "code_challenge_method=S256") {
			t.Errorf("expected S256 in redirect, got %q", location)
		}
	})
}

func TestHandleProviderCallback(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/oauth/providers/github/callback", nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/unknown/callback?code=x&state=y", nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "unknown")
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("error param from provider", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.Providers = map[string]ProviderConfig{
			"github": {Name: "GitHub"},
		}
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?error=access_denied&error_description=user+denied", nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")
		// renderError renders a page — status is 400.
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing code or state", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.Providers = map[string]ProviderConfig{
			"github": {Name: "GitHub"},
		}
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?code=abc", nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid state", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.Providers = map[string]ProviderConfig{
			"github": {Name: "GitHub"},
		}
		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?code=abc&state=nonexistent", nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("expired state", func(t *testing.T) {
		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		oauthStore := NewStore(authStore.DB())
		cfg := testServerConfig()
		cfg.Providers = map[string]ProviderConfig{
			"github": {Name: "GitHub"},
		}
		server := NewServer(cfg, oauthStore, authStore)
		// Clock returns time in the future so state is expired.
		server.clock = func() time.Time { return time.Now().UTC().Add(2 * time.Hour) }

		state, err := oauthStore.CreateProviderState("github", "", "verifier", 10*time.Minute)
		if err != nil {
			t.Fatalf("create state: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?code=abc&state="+state.State, nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "expired") {
			t.Errorf("expected 'expired' in body, got %q", w.Body.String())
		}
	})

	t.Run("full callback success with redirect", func(t *testing.T) {
		// Set up fake token and profile endpoints.
		tokenCalled := false
		profileCalled := false
		mux := http.NewServeMux()
		mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
			tokenCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "provider-tok",
				"expires_in":   3600,
			})
		})
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			profileCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":    123,
				"login": "testuser",
				"name":  "Test User",
			})
		})
		providerServer := httptest.NewServer(mux)
		defer providerServer.Close()

		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		oauthStore := NewStore(authStore.DB())
		fixedTime := time.Now().UTC()
		cfg := testServerConfig()
		cfg.LoginRedirectAllowlist = []string{"http://app.example.com/done"}
		cfg.Providers = map[string]ProviderConfig{
			"github": {
				Name:        "GitHub",
				TokenURL:    providerServer.URL + "/token",
				UserInfoURL: providerServer.URL + "/user",
				RedirectURI: "http://localhost/cb",
			},
		}
		server := NewServer(cfg, oauthStore, authStore)
		server.clock = func() time.Time { return fixedTime }
		server.httpClient = providerServer.Client()

		state, err := oauthStore.CreateProviderState("github", "http://app.example.com/done", "verifier", 10*time.Minute)
		if err != nil {
			t.Fatalf("create state: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?code=abc&state="+state.State, nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")

		if !tokenCalled {
			t.Error("expected token endpoint to be called")
		}
		if !profileCalled {
			t.Error("expected profile endpoint to be called")
		}
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if !strings.HasPrefix(location, "http://app.example.com/done") {
			t.Errorf("expected redirect to app, got %q", location)
		}
		if !strings.Contains(location, "provider=github") {
			t.Errorf("expected provider in redirect, got %q", location)
		}
		if !strings.Contains(location, "provider_user_id=github-123") {
			t.Errorf("expected provider_user_id in redirect, got %q", location)
		}
	})

	t.Run("full callback success without redirect returns JSON", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "provider-tok",
			})
		})
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":    456,
				"login": "jsonuser",
			})
		})
		providerServer := httptest.NewServer(mux)
		defer providerServer.Close()

		path := t.TempDir() + "/auth.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		oauthStore := NewStore(authStore.DB())
		fixedTime := time.Now().UTC()
		cfg := testServerConfig()
		cfg.Providers = map[string]ProviderConfig{
			"github": {
				Name:        "GitHub",
				TokenURL:    providerServer.URL + "/token",
				UserInfoURL: providerServer.URL + "/user",
				RedirectURI: "http://localhost/cb",
			},
		}
		server := NewServer(cfg, oauthStore, authStore)
		server.clock = func() time.Time { return fixedTime }
		server.httpClient = providerServer.Client()

		// No redirect URI in state.
		state, err := oauthStore.CreateProviderState("github", "", "verifier", 10*time.Minute)
		if err != nil {
			t.Fatalf("create state: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/oauth/providers/github/callback?code=abc&state="+state.State, nil)
		w := httptest.NewRecorder()
		server.handleProviderCallback(w, req, "github")

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body["provider"] != "github" {
			t.Errorf("expected provider=github, got %q", body["provider"])
		}
		if body["provider_user_id"] != "github-456" {
			t.Errorf("expected provider_user_id=github-456, got %q", body["provider_user_id"])
		}
		if body["user_id"] == "" {
			t.Error("expected non-empty user_id")
		}
	})
}
