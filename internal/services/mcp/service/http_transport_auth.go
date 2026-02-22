package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// validateLocalRequest enforces host access to mitigate DNS rebinding.
// It checks Host and Origin headers against allowed hosts per MCP guidance so
// remote web pages cannot reach local MCP servers via rebinding.
// This is the transport-side "network guardrail" before we have richer auth.
func (t *HTTPTransport) validateLocalRequest(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("invalid request")
	}

	if !t.isAllowedHostHeader(r.Host) {
		return fmt.Errorf("invalid host")
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin")
	}

	originHost := parsed.Host
	if originHost == "" {
		return fmt.Errorf("invalid origin")
	}

	if !t.isAllowedHostHeader(originHost) {
		return fmt.Errorf("invalid origin")
	}

	return nil
}

// isAllowedHostHeader reports whether a Host/Origin header resolves to an allowed host.
// The default posture is local-only unless explicit hosts are configured.
func (t *HTTPTransport) isAllowedHostHeader(host string) bool {
	resolvedHost, ok := normalizeHost(host)
	if !ok {
		return false
	}

	if isLoopbackHost(resolvedHost) {
		return true
	}

	allowed := t.allowedHosts
	if len(allowed) == 0 {
		return false
	}

	_, ok = allowed[strings.ToLower(resolvedHost)]
	return ok
}

// isLoopbackHost reports whether a host resolves to loopback.
// It is intentionally strict: only explicit local loopback hosts pass by default.
func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

// parseAllowedHosts parses allowed hosts from env-loaded values.
func parseAllowedHosts(hosts []string) map[string]struct{} {
	result := make(map[string]struct{}, len(hosts))
	for _, entry := range hosts {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		result[strings.ToLower(trimmed)] = struct{}{}
	}
	return result
}

// normalizeHost extracts the hostname portion from Host/Origin headers.
func normalizeHost(host string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", false
	}

	if strings.HasPrefix(host, "[") {
		if splitHost, _, err := net.SplitHostPort(host); err == nil {
			return splitHost, true
		}
		if strings.HasSuffix(host, "]") {
			return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"), true
		}
		return "", false
	}

	if strings.Count(host, ":") > 1 {
		return host, true
	}

	if strings.Contains(host, ":") {
		splitHost, _, err := net.SplitHostPort(host)
		if err != nil {
			return "", false
		}
		return splitHost, true
	}

	return host, true
}

// handleHealth handles GET /mcp/health for health checks.
func (t *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

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
	if t.oauth == nil {
		return true
	}
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.writeUnauthorized(w, r, "authorization required")
		return false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		t.writeUnauthorized(w, r, "authorization required")
		return false
	}
	active, err := t.oauth.validateToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, errOAuthResourceSecretMissing) {
			http.Error(w, "oauth introspection misconfigured", http.StatusInternalServerError)
			return false
		}
		t.writeUnauthorized(w, r, "invalid access token")
		return false
	}
	if !active {
		t.writeUnauthorized(w, r, "invalid access token")
		return false
	}
	return true
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
