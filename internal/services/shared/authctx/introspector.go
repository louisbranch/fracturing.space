package authctx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IntrospectionResult mirrors the auth service introspection JSON response.
type IntrospectionResult struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
}

// Introspector validates an OAuth access token via introspection.
type Introspector interface {
	Introspect(ctx context.Context, token string) (IntrospectionResult, error)
}

// HTTPIntrospector calls a remote HTTP introspect endpoint.
type HTTPIntrospector struct {
	url            string
	resourceSecret string
	client         *http.Client
}

// NewHTTPIntrospector creates an introspector that POSTs to the given URL.
func NewHTTPIntrospector(url, resourceSecret string, client *http.Client) *HTTPIntrospector {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPIntrospector{
		url:            url,
		resourceSecret: resourceSecret,
		client:         client,
	}
}

// Introspect validates the token by calling the introspect endpoint.
func (h *HTTPIntrospector) Introspect(ctx context.Context, token string) (IntrospectionResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, nil)
	if err != nil {
		return IntrospectionResult{}, fmt.Errorf("build introspect request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if h.resourceSecret != "" {
		req.Header.Set("X-Resource-Secret", h.resourceSecret)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return IntrospectionResult{}, fmt.Errorf("introspect request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return IntrospectionResult{}, fmt.Errorf("introspect returned %s", resp.Status)
	}

	var result IntrospectionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return IntrospectionResult{}, fmt.Errorf("decode introspect response: %w", err)
	}
	return result, nil
}
