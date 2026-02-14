package oauth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"golang.org/x/crypto/bcrypt"
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

// seedUser creates a credentialed user for testing.
func seedUser(t *testing.T, authStore *authsqlite.Store, oauthStore *Store, username, password string) string {
	t.Helper()
	created, err := user.CreateUser(user.CreateUserInput{DisplayName: "Test User"}, time.Now, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := authStore.PutUser(context.Background(), created); err != nil {
		t.Fatalf("store user: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := oauthStore.UpsertOAuthUserCredentials(created.ID, username, string(hash), time.Now().UTC()); err != nil {
		t.Fatalf("store credentials: %v", err)
	}
	return created.ID
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

func TestHandleLogin(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server, _ := testServer(t)
		req := httptest.NewRequest(http.MethodGet, "/authorize/login", nil)
		w := httptest.NewRecorder()
		server.handleLogin(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid pending_id", func(t *testing.T) {
		server, _ := testServer(t)
		form := url.Values{"pending_id": {"nonexistent"}, "username": {"user"}, "password": {"pass"}}
		req := httptest.NewRequest(http.MethodPost, "/authorize/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleLogin(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong credentials", func(t *testing.T) {
		server, oauthStore := testServer(t)

		// Get auth store to seed user
		path := t.TempDir() + "/auth2.db"
		authStore, err := authsqlite.Open(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { authStore.Close() })

		// Use the server's store directly
		server.store = oauthStore
		server.userStore = authStore

		// Create a pending authorization
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

		form := url.Values{"pending_id": {pendingID}, "username": {"nobody"}, "password": {"wrong"}}
		req := httptest.NewRequest(http.MethodPost, "/authorize/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.handleLogin(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 (re-rendered login), got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "invalid username or password") {
			t.Error("expected error message in re-rendered login form")
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
			ID:          "user-1",
			DisplayName: "Alice",
			Locale:      platformi18n.DefaultLocale(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
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
		if !strings.Contains(w.Body.String(), "Alice") {
			t.Errorf("expected display name")
		}
	})
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

func TestFullAuthorizeLoginConsentFlow(t *testing.T) {
	path := t.TempDir() + "/auth.db"
	authStore, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })

	oauthStore := NewStore(authStore.DB())
	server := NewServer(testServerConfig(), oauthStore, authStore)
	server.config.LoginUIURL = "http://web.local/login"
	userID := seedUser(t, authStore, oauthStore, "testuser", "testpass")
	_ = userID

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := ComputeS256Challenge(codeVerifier)

	// Step 1: Authorize
	authorizeURL, _ := url.Parse(httpServer.URL + "/authorize")
	q := authorizeURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", "test-client")
	q.Set("redirect_uri", "http://localhost:5555/callback")
	q.Set("scope", "openid")
	q.Set("state", "test-state")
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	authorizeURL.RawQuery = q.Encode()

	authorizeClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := authorizeClient.Get(authorizeURL.String())
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d", resp.StatusCode)
	}
	redirected, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	pendingID := redirected.Query().Get("pending_id")
	if pendingID == "" {
		t.Fatalf("missing pending_id in login redirect")
	}

	// Step 2: Login
	loginForm := url.Values{
		"pending_id": {pendingID},
		"username":   {"testuser"},
		"password":   {"testpass"},
	}
	loginResp, err := http.PostForm(httpServer.URL+"/authorize/login", loginForm)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", loginResp.StatusCode)
	}
	consentHTML := readBodyHelper(t, loginResp)
	consentPendingID := extractPendingIDHelper(t, consentHTML)

	// Step 3: Consent (allow)
	consentForm := url.Values{
		"pending_id": {consentPendingID},
		"decision":   {"allow"},
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	consentResp, err := client.PostForm(httpServer.URL+"/authorize/consent", consentForm)
	if err != nil {
		t.Fatalf("consent: %v", err)
	}
	defer consentResp.Body.Close()
	if consentResp.StatusCode != http.StatusFound {
		t.Fatalf("consent status = %d", consentResp.StatusCode)
	}
	location, err := url.Parse(consentResp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatal("expected code in redirect")
	}
	if location.Query().Get("state") != "test-state" {
		t.Errorf("expected state test-state, got %q", location.Query().Get("state"))
	}

	// Step 4: Token exchange
	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"http://localhost:5555/callback"},
		"code_verifier": {codeVerifier},
		"client_id":     {"test-client"},
	}
	tokenResp, err := http.PostForm(httpServer.URL+"/token", tokenForm)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d", tokenResp.StatusCode)
	}
	var tokenBody tokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenBody); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if tokenBody.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}

	// Step 5: Introspect
	introspectReq, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/introspect", nil)
	introspectReq.Header.Set("Authorization", "Bearer "+tokenBody.AccessToken)
	introspectReq.Header.Set("X-Resource-Secret", "test-resource-secret")
	introspectResp, err := http.DefaultClient.Do(introspectReq)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}
	defer introspectResp.Body.Close()
	var introspectBody introspectResponse
	if err := json.NewDecoder(introspectResp.Body).Decode(&introspectBody); err != nil {
		t.Fatalf("decode introspect: %v", err)
	}
	if !introspectBody.Active {
		t.Fatal("expected active token")
	}
}

func TestHandleLogin_ExpiredPending(t *testing.T) {
	server, oauthStore := testServer(t)
	// Create a pending authorization with a very short TTL so it expires immediately.
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
	// Wait for expiry.
	time.Sleep(2 * time.Millisecond)

	form := url.Values{"pending_id": {pendingID}, "username": {"user"}, "password": {"pass"}}
	req := httptest.NewRequest(http.MethodPost, "/authorize/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleLogin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for expired pending, got %d", w.Code)
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	path := t.TempDir() + "/auth.db"
	authStore, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })

	oauthStore := NewStore(authStore.DB())
	server := NewServer(testServerConfig(), oauthStore, authStore)
	_ = seedUser(t, authStore, oauthStore, "testuser", "correctpass")

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

	form := url.Values{"pending_id": {pendingID}, "username": {"testuser"}, "password": {"wrongpass"}}
	req := httptest.NewRequest(http.MethodPost, "/authorize/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.handleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (re-rendered login), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid username or password") {
		t.Error("expected error message in login form")
	}
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

func readBodyHelper(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}

func extractPendingIDHelper(t *testing.T, html string) string {
	t.Helper()
	re := regexp.MustCompile(`name="pending_id" value="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		t.Fatalf("pending_id not found in HTML:\n%s", html[:min(len(html), 500)])
	}
	return matches[1]
}
