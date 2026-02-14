package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// testServerConfig returns a minimal test configuration with one registered client.
func testServerConfig() Config {
	return Config{
		ResourceSecret: "test-resource-secret",
		Clients: []Client{
			{
				ID:                      "test-client",
				RedirectURIs:            []string{"http://localhost:5555/callback"},
				Name:                    "Test Client",
				TokenEndpointAuthMethod: "none",
			},
		},
		AuthorizationCodeTTL:    10 * time.Minute,
		PendingAuthorizationTTL: 15 * time.Minute,
		TokenTTL:                time.Hour,
	}
}

// testServer creates a fully wired Server backed by an in-memory SQLite store.
func testServer(t *testing.T) (*Server, *Store) {
	t.Helper()
	path := t.TempDir() + "/auth.db"
	authStore, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })

	oauthStore := NewStore(authStore.DB())
	server := NewServer(testServerConfig(), oauthStore, authStore)
	return server, oauthStore
}

func TestHandleAuthorize(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/authorize", nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("unsupported response type", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=token&client_id=test-client", nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unknown client", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=code&client_id=unknown", nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing redirect_uri", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=code&client_id=test-client", nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unregistered redirect_uri", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=code&client_id=test-client&redirect_uri=http://evil.com/cb", nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing code_challenge", func(t *testing.T) {
		server, _ := testServer(t)
		q := url.Values{
			"response_type": {"code"},
			"client_id":     {"test-client"},
			"redirect_uri":  {"http://localhost:5555/callback"},
		}
		req := httptest.NewRequest(http.MethodGet, "/authorize?"+q.Encode(), nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		// Should redirect with error
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
	})

	t.Run("wrong code_challenge_method", func(t *testing.T) {
		server, _ := testServer(t)
		q := url.Values{
			"response_type":         {"code"},
			"client_id":             {"test-client"},
			"redirect_uri":          {"http://localhost:5555/callback"},
			"code_challenge":        {"abc123"},
			"code_challenge_method": {"plain"},
		}
		req := httptest.NewRequest(http.MethodGet, "/authorize?"+q.Encode(), nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
	})

	t.Run("success redirects to login ui when configured", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.LoginUIURL = "http://web.local/login"
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)
		q := url.Values{
			"response_type":         {"code"},
			"client_id":             {"test-client"},
			"redirect_uri":          {"http://localhost:5555/callback"},
			"code_challenge":        {codeChallenge},
			"code_challenge_method": {"S256"},
			"state":                 {"test-state"},
		}
		req := httptest.NewRequest(http.MethodGet, "/authorize?"+q.Encode(), nil)
		w := httptest.NewRecorder()
		server.handleAuthorize(w, req)
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		redirected, err := url.Parse(location)
		if err != nil {
			t.Fatalf("parse redirect: %v", err)
		}
		if redirected.Host != "web.local" {
			t.Fatalf("redirect host = %q, want %q", redirected.Host, "web.local")
		}
		if redirected.Path != "/login" {
			t.Fatalf("redirect path = %q, want %q", redirected.Path, "/login")
		}
		if redirected.Query().Get("pending_id") == "" {
			t.Fatal("expected pending_id in redirect query")
		}
		if redirected.Query().Get("client_id") != "test-client" {
			t.Fatalf("client_id = %q, want %q", redirected.Query().Get("client_id"), "test-client")
		}
		if redirected.Query().Get("client_name") != "Test Client" {
			t.Fatalf("client_name = %q, want %q", redirected.Query().Get("client_name"), "Test Client")
		}
	})
}

func TestHandleConsent(t *testing.T) {
	t.Run("missing pending id", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize/consent", nil)
		w := httptest.NewRecorder()
		server.handleConsent(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid pending_id", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{"pending_id": {"nonexistent"}, "decision": {"allow"}}
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleConsent(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("deny redirects with error", func(t *testing.T) {
		server, oauthStore := testServer(t)
		pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       "test-challenge",
			CodeChallengeMethod: "S256",
			State:               "some-state",
		}, 15*time.Minute)
		if err != nil {
			t.Fatalf("create pending: %v", err)
		}
		// Set user ID on pending auth
		if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, "user-1"); err != nil {
			t.Fatalf("update pending: %v", err)
		}

		form := url.Values{"pending_id": {pendingID}, "decision": {"deny"}}
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleConsent(w, req)
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if !strings.Contains(location, "error=access_denied") {
			t.Errorf("expected access_denied in redirect, got %q", location)
		}
	})

	t.Run("allow redirects with code", func(t *testing.T) {
		server, oauthStore := testServer(t)
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)

		pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
			State:               "my-state",
		}, 15*time.Minute)
		if err != nil {
			t.Fatalf("create pending: %v", err)
		}
		if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, "user-1"); err != nil {
			t.Fatalf("update pending: %v", err)
		}

		form := url.Values{"pending_id": {pendingID}, "decision": {"allow"}}
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleConsent(w, req)
		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if !strings.Contains(location, "code=") {
			t.Errorf("expected code in redirect, got %q", location)
		}
		if !strings.Contains(location, "state=my-state") {
			t.Errorf("expected state in redirect, got %q", location)
		}
	})

	t.Run("get renders consent view", func(t *testing.T) {
		server, oauthStore := testServer(t)
		if err := server.userStore.PutUser(context.Background(), user.User{
			ID:        "user-1",
			Username:  "alice",
			Locale:    platformi18n.DefaultLocale(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("put user: %v", err)
		}

		pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       "test-challenge",
			CodeChallengeMethod: "S256",
		}, 15*time.Minute)
		if err != nil {
			t.Fatalf("create pending: %v", err)
		}
		if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, "user-1"); err != nil {
			t.Fatalf("update pending: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/authorize/consent?pending_id="+pendingID, nil)
		w := httptest.NewRecorder()
		server.handleConsent(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Signed in as") {
			t.Errorf("expected consent view")
		}
		if !strings.Contains(w.Body.String(), "alice") {
			t.Errorf("expected username")
		}
	})
}

func TestHandleConsent_AutoApprovesTrustedClient(t *testing.T) {
	server, oauthStore := testServer(t)
	// Mark the test-client as trusted.
	server.config.Clients = []Client{
		{
			ID:                      "test-client",
			RedirectURIs:            []string{"http://localhost:5555/callback"},
			Name:                    "Test Client",
			TokenEndpointAuthMethod: "none",
			Trusted:                 true,
		},
	}

	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := ComputeS256Challenge(codeVerifier)

	pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		State:               "my-state",
	}, 15*time.Minute)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}
	if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, "user-1"); err != nil {
		t.Fatalf("update pending: %v", err)
	}

	// GET consent for a trusted client should auto-approve (redirect with code).
	req := httptest.NewRequest(http.MethodGet, "/authorize/consent?pending_id="+pendingID, nil)
	w := httptest.NewRecorder()
	server.handleConsent(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "code=") {
		t.Fatalf("expected code in redirect, got %q", location)
	}
	if !strings.Contains(location, "state=my-state") {
		t.Fatalf("expected state in redirect, got %q", location)
	}
}

func TestHandleConsent_NonTrustedClientRendersConsentView(t *testing.T) {
	server, oauthStore := testServer(t)
	// Ensure client is NOT trusted (default).
	if err := server.userStore.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Username:  "alice",
		Locale:    platformi18n.DefaultLocale(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
	}, 15*time.Minute)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}
	if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, "user-1"); err != nil {
		t.Fatalf("update pending: %v", err)
	}

	// GET consent for non-trusted client should render consent form (200).
	req := httptest.NewRequest(http.MethodGet, "/authorize/consent?pending_id="+pendingID, nil)
	w := httptest.NewRecorder()
	server.handleConsent(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Signed in as") {
		t.Fatal("expected consent view to render")
	}
}

func TestHandleToken(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/token", nil)
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("unsupported grant type", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{"grant_type": {"implicit"}}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{"grant_type": {"authorization_code"}}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unknown client", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"some-code"},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
			"client_id":     {"unknown-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid authorization code", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"bad-code"},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
			"client_id":     {"test-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		server, oauthStore := testServer(t)
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)

		authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
			Scope:               "openid",
		}, "user-1", 10*time.Minute)
		if err != nil {
			t.Fatalf("create auth code: %v", err)
		}

		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode.Code},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {codeVerifier},
			"client_id":     {"test-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp tokenResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.AccessToken == "" {
			t.Error("expected non-empty access token")
		}
		if resp.TokenType != "Bearer" {
			t.Errorf("expected token type Bearer, got %q", resp.TokenType)
		}
	})

	t.Run("PKCE mismatch", func(t *testing.T) {
		server, oauthStore := testServer(t)
		codeChallenge := ComputeS256Challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")

		authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
		}, "user-1", 10*time.Minute)
		if err != nil {
			t.Fatalf("create auth code: %v", err)
		}

		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode.Code},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {"wrong-verifier-that-is-long-enough-for-the-min-length-check-aa"},
			"client_id":     {"test-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		var errResp errorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if errResp.Error != "invalid_grant" {
			t.Errorf("expected invalid_grant, got %q", errResp.Error)
		}
	})

	t.Run("redirect_uri mismatch", func(t *testing.T) {
		server, oauthStore := testServer(t)
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)

		authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
		}, "user-1", 10*time.Minute)
		if err != nil {
			t.Fatalf("create auth code: %v", err)
		}

		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode.Code},
			"redirect_uri":  {"http://different-host/callback"},
			"code_verifier": {codeVerifier},
			"client_id":     {"test-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("client_id mismatch", func(t *testing.T) {
		server, oauthStore := testServer(t)
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)

		// Add a second client
		server.config.Clients = append(server.config.Clients, Client{
			ID:                      "other-client",
			RedirectURIs:            []string{"http://localhost:5555/callback"},
			TokenEndpointAuthMethod: "none",
		})

		authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
		}, "user-1", 10*time.Minute)
		if err != nil {
			t.Fatalf("create auth code: %v", err)
		}

		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode.Code},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {codeVerifier},
			"client_id":     {"other-client"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestHandleIntrospect(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/introspect", nil)
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("missing resource secret config", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.ResourceSecret = ""
		req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("wrong resource secret", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
		req.Header.Set("X-Resource-Secret", "wrong-secret")
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("missing bearer token", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
		req.Header.Set("X-Resource-Secret", "test-resource-secret")
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid token returns inactive", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
		req.Header.Set("X-Resource-Secret", "test-resource-secret")
		req.Header.Set("Authorization", "Bearer bad-token")
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var resp introspectResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Active {
			t.Error("expected inactive token")
		}
	})

	t.Run("valid token returns active", func(t *testing.T) {
		server, oauthStore := testServer(t)
		token, err := oauthStore.CreateAccessToken("test-client", "user-1", "openid", time.Hour)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
		req.Header.Set("X-Resource-Secret", "test-resource-secret")
		req.Header.Set("Authorization", "Bearer "+token.Token)
		w := httptest.NewRecorder()
		server.handleIntrospect(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var resp introspectResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Active {
			t.Error("expected active token")
		}
		if resp.ClientID != "test-client" {
			t.Errorf("expected client_id %q, got %q", "test-client", resp.ClientID)
		}
		if resp.UserID != "user-1" {
			t.Errorf("expected user_id %q, got %q", "user-1", resp.UserID)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("clientDisplayName nil", func(t *testing.T) {
		if got := clientDisplayName(nil); got != "Unknown Client" {
			t.Errorf("expected %q, got %q", "Unknown Client", got)
		}
	})

	t.Run("clientDisplayName with name", func(t *testing.T) {
		if got := clientDisplayName(&Client{ID: "id", Name: "My App"}); got != "My App" {
			t.Errorf("expected %q, got %q", "My App", got)
		}
	})

	t.Run("clientDisplayName without name", func(t *testing.T) {
		if got := clientDisplayName(&Client{ID: "my-client"}); got != "my-client" {
			t.Errorf("expected %q, got %q", "my-client", got)
		}
	})

	t.Run("redirectURIAllowed", func(t *testing.T) {
		allowed := []string{"http://a.com/cb", "http://b.com/cb"}
		if !redirectURIAllowed("http://a.com/cb", allowed) {
			t.Error("expected allowed")
		}
		if redirectURIAllowed("http://c.com/cb", allowed) {
			t.Error("expected rejected")
		}
		if redirectURIAllowed("http://a.com/cb", nil) {
			t.Error("expected rejected for nil list")
		}
	})

	t.Run("formatScopes", func(t *testing.T) {
		if got := formatScopes("openid profile"); len(got) != 2 {
			t.Errorf("expected 2 scopes, got %d", len(got))
		}
		if got := formatScopes(""); got[0] != "basic profile" {
			t.Errorf("expected default scope, got %v", got)
		}
	})

	t.Run("validateTokenClientAuth", func(t *testing.T) {
		// nil client
		if err := validateTokenClientAuth(nil, ""); err == nil {
			t.Error("expected error for nil client")
		}
		// no auth method, no secret -> "none"
		if err := validateTokenClientAuth(&Client{ID: "c"}, ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// client_secret_post with matching secret
		if err := validateTokenClientAuth(&Client{ID: "c", Secret: "s", TokenEndpointAuthMethod: "client_secret_post"}, "s"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// client_secret_post with wrong secret
		if err := validateTokenClientAuth(&Client{ID: "c", Secret: "s", TokenEndpointAuthMethod: "client_secret_post"}, "wrong"); err == nil {
			t.Error("expected error for wrong secret")
		}
		// unsupported method
		if err := validateTokenClientAuth(&Client{ID: "c", TokenEndpointAuthMethod: "private_key_jwt"}, ""); err == nil {
			t.Error("expected error for unsupported method")
		}
	})
}

func TestRegisterRoutes(t *testing.T) {
	t.Run("nil mux is safe", func(t *testing.T) {
		server, _ := testServer(t)
		server.RegisterRoutes(nil)
	})

	t.Run("registers routes", func(t *testing.T) {
		server, _ := testServer(t)
		mux := http.NewServeMux()
		server.RegisterRoutes(mux)

		// Test the health endpoint
		req := httptest.NewRequest(http.MethodGet, "/up", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestStartCleanup(t *testing.T) {
	t.Run("nil server is safe", func(t *testing.T) {
		var s *Server
		s.StartCleanup(context.Background(), time.Minute)
	})

	t.Run("nil store is safe", func(t *testing.T) {
		s := &Server{}
		s.StartCleanup(context.Background(), time.Minute)
	})

	t.Run("zero interval is safe", func(t *testing.T) {
		server, _ := testServer(t)
		server.StartCleanup(context.Background(), 0)
	})

	t.Run("starts and stops", func(t *testing.T) {
		server, _ := testServer(t)
		ctx, cancel := context.WithCancel(context.Background())
		server.StartCleanup(ctx, 10*time.Millisecond)
		time.Sleep(50 * time.Millisecond)
		cancel()
	})
}

func TestHandleConsent_ExpiredPending(t *testing.T) {
	server, oauthStore := testServer(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
	}, 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}
	time.Sleep(2 * time.Millisecond)

	form := url.Values{"pending_id": {pendingID}, "decision": {"allow"}}
	req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleConsent(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for expired pending, got %d", w.Code)
	}
}

func TestHandleConsent_UserNotAuthenticated(t *testing.T) {
	server, oauthStore := testServer(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
	}, 15*time.Minute)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}
	// Don't set user ID — pending.UserID is empty.

	form := url.Values{"pending_id": {pendingID}, "decision": {"allow"}}
	req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleConsent(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unauthenticated user, got %d", w.Code)
	}
}

func TestHandleToken_ExpiredCode(t *testing.T) {
	server, oauthStore := testServer(t)
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := ComputeS256Challenge(codeVerifier)

	authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
	}, "user-1", 1*time.Nanosecond) // Expires immediately
	if err != nil {
		t.Fatalf("create auth code: %v", err)
	}
	time.Sleep(2 * time.Millisecond)

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode.Code},
		"redirect_uri":  {"http://localhost:5555/callback"},
		"code_verifier": {codeVerifier},
		"client_id":     {"test-client"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleToken(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for expired code, got %d", w.Code)
	}
	var errResp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %q", errResp.Error)
	}
}

func TestHandleToken_CodeAlreadyUsed(t *testing.T) {
	server, oauthStore := testServer(t)
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := ComputeS256Challenge(codeVerifier)

	authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
	}, "user-1", 10*time.Minute)
	if err != nil {
		t.Fatalf("create auth code: %v", err)
	}
	// Mark it as used.
	if _, err := oauthStore.MarkAuthorizationCodeUsed(authCode.Code); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode.Code},
		"redirect_uri":  {"http://localhost:5555/callback"},
		"code_verifier": {codeVerifier},
		"client_id":     {"test-client"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleToken(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for used code, got %d", w.Code)
	}
	var errResp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %q", errResp.Error)
	}
}

func TestHandleToken_ClientSecretAuth(t *testing.T) {
	t.Run("valid secret", func(t *testing.T) {
		server, oauthStore := testServer(t)
		server.config.Clients = []Client{
			{
				ID:                      "secret-client",
				Secret:                  "my-secret",
				RedirectURIs:            []string{"http://localhost:5555/callback"},
				TokenEndpointAuthMethod: "client_secret_post",
			},
		}
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := ComputeS256Challenge(codeVerifier)

		authCode, err := oauthStore.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "secret-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: "S256",
		}, "user-1", 10*time.Minute)
		if err != nil {
			t.Fatalf("create auth code: %v", err)
		}

		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode.Code},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {codeVerifier},
			"client_id":     {"secret-client"},
			"client_secret": {"my-secret"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		server, _ := testServer(t)
		server.config.Clients = []Client{
			{
				ID:                      "secret-client",
				Secret:                  "my-secret",
				RedirectURIs:            []string{"http://localhost:5555/callback"},
				TokenEndpointAuthMethod: "client_secret_post",
			},
		}
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"some-code"},
			"redirect_uri":  {"http://localhost:5555/callback"},
			"code_verifier": {"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
			"client_id":     {"secret-client"},
			"client_secret": {"wrong-secret"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleToken(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
}

func TestHandleIntrospect_EmptyBearerToken(t *testing.T) {
	server, _ := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/introspect", nil)
	req.Header.Set("X-Resource-Secret", "test-resource-secret")
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	server.handleIntrospect(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty bearer token, got %d", w.Code)
	}
}
