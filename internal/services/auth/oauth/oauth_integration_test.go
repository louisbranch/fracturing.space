//go:build integration

package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
)

// TestOAuthAuthorizationCodeFlow exercises the authorize → consent → token → introspect
// flow end-to-end. Since password login has been removed, the test directly sets the
// pending authorization's user ID to simulate authentication (e.g. via passkey or
// external provider).
func TestOAuthAuthorizationCodeFlow(t *testing.T) {
	store := openAuthStoreForTest(t)
	oauthStore := NewStore(store.DB())

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

	// Step 1: /authorize → redirects to login UI with pending_id.
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

	// Step 2: Simulate authentication by setting the user ID on the pending authorization.
	userID := "integration-user-1"
	if err := oauthStore.UpdatePendingAuthorizationUserID(pendingID, userID); err != nil {
		t.Fatalf("set pending user id: %v", err)
	}

	// Step 3: Consent (allow).
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

	// Step 4: Token exchange.
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

	// Step 5: Introspect.
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

	// Step 6: Invalid token returns inactive.
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
