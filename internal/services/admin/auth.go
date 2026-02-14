package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// tokenCookieName is the domain-scoped cookie set by the web login service.
const tokenCookieName = "fs_token"

// AuthConfig holds the settings for admin authentication middleware.
type AuthConfig struct {
	IntrospectURL  string
	ResourceSecret string
	LoginURL       string
}

// requireAuth wraps next with token-introspection-based authentication.
// Requests to auth-exempt paths (e.g. /static/) pass through unchecked.
func requireAuth(next http.Handler, introspector TokenIntrospector, loginURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAuthExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(tokenCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		result, err := introspector.Introspect(r.Context(), cookie.Value)
		if err != nil {
			log.Printf("admin auth introspect error: %v", err)
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}
		if !result.Active {
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		ctx := contextWithAuthUser(r.Context(), result.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isAuthExempt returns true for paths that should bypass authentication.
func isAuthExempt(path string) bool {
	return strings.HasPrefix(path, "/static/")
}

// TokenIntrospector validates an OAuth access token via introspection.
type TokenIntrospector interface {
	Introspect(ctx context.Context, token string) (introspectResponse, error)
}

// introspectResponse mirrors the auth service's introspect JSON shape.
type introspectResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
}

// httpIntrospector calls a remote HTTP introspect endpoint.
type httpIntrospector struct {
	url            string
	resourceSecret string
	client         *http.Client
}

// newHTTPIntrospector creates an introspector that POSTs to the given URL.
func newHTTPIntrospector(url, resourceSecret string) *httpIntrospector {
	return &httpIntrospector{
		url:            url,
		resourceSecret: resourceSecret,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Introspect validates the token by calling the introspect endpoint.
func (h *httpIntrospector) Introspect(ctx context.Context, token string) (introspectResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, nil)
	if err != nil {
		return introspectResponse{}, fmt.Errorf("build introspect request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if h.resourceSecret != "" {
		req.Header.Set("X-Resource-Secret", h.resourceSecret)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return introspectResponse{}, fmt.Errorf("introspect request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return introspectResponse{}, fmt.Errorf("introspect returned %s", resp.Status)
	}

	var result introspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return introspectResponse{}, fmt.Errorf("decode introspect response: %w", err)
	}
	return result, nil
}
