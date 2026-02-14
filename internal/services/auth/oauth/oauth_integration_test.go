//go:build integration

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

	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"golang.org/x/crypto/bcrypt"
)

func TestOAuthAuthorizationCodeFlow(t *testing.T) {
	store := openAuthStoreForTest(t)
	oauthStore := NewStore(store.DB())
	username := "demo-user"
	password := "s3cret-pass"
	userID := seedOAuthUser(t, store, oauthStore, username, password)

	redirectURI := "http://localhost:5555/callback"
	config := Config{
		ResourceSecret:          "resource-secret",
		Clients:                 []Client{{ID: "client-1", RedirectURIs: []string{redirectURI}, Name: "Test Client", TokenEndpointAuthMethod: "none"}},
		AuthorizationCodeTTL:    10 * time.Minute,
		PendingAuthorizationTTL: 15 * time.Minute,
		TokenTTL:                time.Hour,
		LoginUIURL:              "http://web.local/login",
	}
	server := NewServer(config, oauthStore, store)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := ComputeS256Challenge(codeVerifier)
	state := "state-123"

	authorizeURL, err := url.Parse(httpServer.URL + "/authorize")
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	query := authorizeURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", "client-1")
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", "openid profile")
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	authorizeURL.RawQuery = query.Encode()

	authorizeClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := authorizeClient.Get(authorizeURL.String())
	if err != nil {
		t.Fatalf("authorize request: %v", err)
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

	loginForm := url.Values{}
	loginForm.Set("pending_id", pendingID)
	loginForm.Set("username", username)
	loginForm.Set("password", password)
	loginResp, err := http.PostForm(httpServer.URL+"/authorize/login", loginForm)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", loginResp.StatusCode)
	}
	consentHTML := readBody(t, loginResp)
	consentPending := extractPendingID(t, consentHTML)
	if consentPending != pendingID {
		t.Fatalf("pending id mismatch: %s != %s", consentPending, pendingID)
	}

	consentForm := url.Values{}
	consentForm.Set("pending_id", pendingID)
	consentForm.Set("decision", "allow")
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	consentReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/authorize/consent", strings.NewReader(consentForm.Encode()))
	if err != nil {
		t.Fatalf("consent request: %v", err)
	}
	consentReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	consentResp, err := client.Do(consentReq)
	if err != nil {
		t.Fatalf("consent request: %v", err)
	}
	defer consentResp.Body.Close()
	if consentResp.StatusCode != http.StatusFound {
		t.Fatalf("consent status = %d", consentResp.StatusCode)
	}
	location := consentResp.Header.Get("Location")
	if location == "" {
		t.Fatalf("missing redirect location")
	}
	redirected, err = url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := redirected.Query().Get("code")
	if code == "" {
		t.Fatalf("missing authorization code")
	}

	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("redirect_uri", redirectURI)
	tokenForm.Set("client_id", "client-1")
	tokenForm.Set("code_verifier", codeVerifier)
	tokenResp, err := http.PostForm(httpServer.URL+"/token", tokenForm)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d", tokenResp.StatusCode)
	}
	var tokenPayload tokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if tokenPayload.AccessToken == "" {
		t.Fatalf("access_token missing")
	}

	introspectReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/introspect", nil)
	if err != nil {
		t.Fatalf("introspect request: %v", err)
	}
	introspectReq.Header.Set("Authorization", "Bearer "+tokenPayload.AccessToken)
	introspectReq.Header.Set("X-Resource-Secret", "resource-secret")
	introspectResp, err := http.DefaultClient.Do(introspectReq)
	if err != nil {
		t.Fatalf("introspect request: %v", err)
	}
	defer introspectResp.Body.Close()
	if introspectResp.StatusCode != http.StatusOK {
		t.Fatalf("introspect status = %d", introspectResp.StatusCode)
	}
	var introspect introspectResponse
	if err := json.NewDecoder(introspectResp.Body).Decode(&introspect); err != nil {
		t.Fatalf("decode introspect: %v", err)
	}
	if !introspect.Active {
		t.Fatalf("introspect inactive token")
	}
	if introspect.UserID != userID {
		t.Fatalf("introspect user_id mismatch: %s != %s", introspect.UserID, userID)
	}

	invalidReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/introspect", nil)
	if err != nil {
		t.Fatalf("introspect invalid request: %v", err)
	}
	invalidReq.Header.Set("Authorization", "Bearer bad-token")
	invalidReq.Header.Set("X-Resource-Secret", "resource-secret")
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("introspect invalid request: %v", err)
	}
	defer invalidResp.Body.Close()
	var invalidPayload introspectResponse
	if err := json.NewDecoder(invalidResp.Body).Decode(&invalidPayload); err != nil {
		t.Fatalf("decode invalid introspect: %v", err)
	}
	if invalidPayload.Active {
		t.Fatalf("expected inactive token")
	}
}

func openAuthStoreForTest(t *testing.T) *authsqlite.Store {
	t.Helper()
	path := t.TempDir() + "/auth.db"
	store, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	return store
}

func seedOAuthUser(t *testing.T, store *authsqlite.Store, oauthStore *Store, username, password string) string {
	t.Helper()
	created, err := user.CreateUser(user.CreateUserInput{DisplayName: "OAuth Tester"}, time.Now, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.PutUser(context.Background(), created); err != nil {
		t.Fatalf("store user: %v", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := oauthStore.UpsertOAuthUserCredentials(created.ID, username, string(passwordHash), time.Now().UTC()); err != nil {
		t.Fatalf("store credentials: %v", err)
	}
	return created.ID
}

func extractPendingID(t *testing.T, html string) string {
	t.Helper()
	re := regexp.MustCompile(`name="pending_id" value="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		t.Fatalf("pending_id not found")
	}
	return matches[1]
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
