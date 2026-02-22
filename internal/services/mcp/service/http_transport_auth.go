package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type oauthAuth struct {
	issuer         string
	resourceSecret string
	httpClient     *http.Client
}

var errOAuthResourceSecretMissing = errors.New("oauth resource secret is not configured")

type protectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
}

type introspectionPayload struct {
	Active bool `json:"active"`
}

// loadOAuthAuthFromEnv builds optional transport-level auth from environment.
// OAuth is optional so MCP can run in trusted local mode without extra
// operational prerequisites.
func loadOAuthAuthFromEnv(raw mcpHTTPEnv) *oauthAuth {
	issuer := strings.TrimSpace(raw.OAuthIssuer)
	if issuer == "" {
		return nil
	}
	return &oauthAuth{
		issuer:         strings.TrimRight(issuer, "/"),
		resourceSecret: raw.OAuthSecret,
		httpClient:     &http.Client{Timeout: defaultIntrospectionTimeout},
	}
}

// authorizeRequest runs bearer-token checks only when OAuth config is present.
// This keeps transport behavior explicit at the boundary while allowing local
// deployments to skip token validation without changing handler wiring.
func (t *HTTPTransport) authorizeRequest(w http.ResponseWriter, r *http.Request) bool {
	if err := t.rateLimitRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return false
	}

	if err := t.authorize(r); err != nil {
		if errors.Is(err, errOAuthResourceSecretMissing) {
			http.Error(w, "oauth introspection misconfigured", http.StatusInternalServerError)
			return false
		}
		t.writeUnauthorized(w, r, err.Error())
		return false
	}
	return true
}

func (t *HTTPTransport) rateLimitRequest(r *http.Request) error {
	if t == nil || t.rateLimiter == nil {
		return nil
	}
	return t.rateLimiter.Allow(r)
}

func (t *HTTPTransport) authorize(r *http.Request) error {
	if t == nil {
		return nil
	}
	if t.requestAuthz != nil {
		return t.requestAuthz.Authorize(r)
	}
	return (&hybridRequestAuthorizer{
		apiToken: t.apiToken,
		oauth:    t.oauth,
	}).Authorize(r)
}

type hybridRequestAuthorizer struct {
	apiToken string
	oauth    *oauthAuth
}

func (a *hybridRequestAuthorizer) Authorize(r *http.Request) error {
	if a == nil {
		return nil
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	apiToken := extractBearerToken(authHeader)
	if apiToken != "" && a.apiToken != "" {
		if subtle.ConstantTimeCompare([]byte(apiToken), []byte(a.apiToken)) == 1 {
			return nil
		}
	}

	if a.oauth == nil {
		if a.apiToken == "" {
			return nil
		}
		return errors.New("authorization required")
	}

	if authHeader == "" {
		return errors.New("authorization required")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return errors.New("authorization required")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return errors.New("authorization required")
	}

	active, err := a.oauth.validateToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, errOAuthResourceSecretMissing) {
			return err
		}
		return errors.New("invalid access token")
	}
	if !active {
		return errors.New("invalid access token")
	}
	return nil
}

func extractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}

// handleProtectedResourceMetadata publishes oauth protected-resource metadata for
// MCP clients that can introspect bearer-token expectations.
func (t *HTTPTransport) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if t.oauth == nil {
		http.NotFound(w, r)
		return
	}
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	baseURL := baseURLFromRequest(r)
	metadata := protectedResourceMetadata{
		Resource:               baseURL + "/mcp",
		AuthorizationServers:   []string{t.oauth.issuer},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

func (t *HTTPTransport) writeUnauthorized(w http.ResponseWriter, r *http.Request, message string) {
	metadataURL := baseURLFromRequest(r) + "/.well-known/oauth-protected-resource"
	w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+metadataURL+`"`)
	http.Error(w, message, http.StatusUnauthorized)
}

// validateToken asks the OAuth resource server whether a token is currently
// active; transport admission is all-or-nothing at MCP call time.
func (a *oauthAuth) validateToken(ctx context.Context, token string) (bool, error) {
	if a == nil || a.issuer == "" {
		return false, errors.New("oauth issuer is not configured")
	}
	if strings.TrimSpace(a.resourceSecret) == "" {
		return false, errOAuthResourceSecretMissing
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.issuer+"/introspect", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Resource-Secret", a.resourceSecret)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, errors.New("introspection failed")
	}
	var payload introspectionPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.Active, nil
}

func baseURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}
