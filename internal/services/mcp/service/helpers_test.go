package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"Localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{" localhost ", true},
		{"example.com", false},
		{"127.0.0.2", false},
		{"", false},
		{"local", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isLoopbackHost(tt.host); got != tt.want {
				t.Errorf("isLoopbackHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input  string
		want   string
		wantOk bool
	}{
		{"localhost:8081", "localhost", true},
		{"example.com:443", "example.com", true},
		{"[::1]:8081", "::1", true},
		{"[::1]", "::1", true},
		{"::1", "::1", true},
		{"example.com", "example.com", true},
		{"", "", false},
		{"  ", "", false},
		{"[::1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := normalizeHost(tt.input)
			if ok != tt.wantOk {
				t.Errorf("normalizeHost(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("normalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteSessionError(t *testing.T) {
	w := httptest.NewRecorder()
	writeSessionError(w, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", body["jsonrpc"])
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["message"] != "test error" {
		t.Errorf("expected message %q, got %v", "test error", errObj["message"])
	}
}

func TestBaseURLFromRequest(t *testing.T) {
	t.Run("plain HTTP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8081/path", nil)
		req.Host = "localhost:8081"
		got := baseURLFromRequest(req)
		if got != "http://localhost:8081" {
			t.Errorf("expected %q, got %q", "http://localhost:8081", got)
		}
	})

	t.Run("with X-Forwarded-Proto", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
		req.Host = "example.com"
		req.Header.Set("X-Forwarded-Proto", "https")
		got := baseURLFromRequest(req)
		if got != "https://example.com" {
			t.Errorf("expected %q, got %q", "https://example.com", got)
		}
	})
}

func TestIsAllowedHostHeader(t *testing.T) {
	t.Run("loopback always allowed", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if !transport.isAllowedHostHeader("localhost:8081") {
			t.Error("expected localhost to be allowed")
		}
		if !transport.isAllowedHostHeader("[::1]:8081") {
			t.Error("expected [::1] to be allowed")
		}
	})

	t.Run("configured host allowed", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		transport.allowedHosts = map[string]struct{}{
			"example.com": {},
		}
		if !transport.isAllowedHostHeader("example.com:443") {
			t.Error("expected example.com to be allowed")
		}
	})

	t.Run("unknown host rejected", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if transport.isAllowedHostHeader("evil.com:8081") {
			t.Error("expected evil.com to be rejected")
		}
	})

	t.Run("empty host rejected", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if transport.isAllowedHostHeader("") {
			t.Error("expected empty host to be rejected")
		}
	})
}

func TestValidateLocalRequest(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if err := transport.validateLocalRequest(nil); err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("valid localhost no origin", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost:8081"
		if err := transport.validateLocalRequest(req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid localhost with origin", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost:8081"
		req.Header.Set("Origin", "http://localhost:8081")
		if err := transport.validateLocalRequest(req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid host", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "evil.com"
		if err := transport.validateLocalRequest(req); err == nil {
			t.Fatal("expected error for invalid host")
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost:8081"
		req.Header.Set("Origin", "http://evil.com")
		if err := transport.validateLocalRequest(req); err == nil {
			t.Fatal("expected error for invalid origin")
		}
	})

	t.Run("malformed origin", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost:8081"
		req.Header.Set("Origin", ":::bad")
		if err := transport.validateLocalRequest(req); err == nil {
			t.Fatal("expected error for malformed origin")
		}
	})
}

func TestOAuthValidateToken(t *testing.T) {
	t.Run("nil auth", func(t *testing.T) {
		var auth *oauthAuth
		active, err := auth.validateToken(context.Background(), "token")
		if err == nil {
			t.Fatal("expected error for nil auth")
		}
		if active {
			t.Error("expected inactive")
		}
	})

	t.Run("missing resource secret", func(t *testing.T) {
		auth := &oauthAuth{
			issuer:     "http://issuer.test",
			httpClient: &http.Client{},
		}
		_, err := auth.validateToken(context.Background(), "token")
		if err != errOAuthResourceSecretMissing {
			t.Errorf("expected errOAuthResourceSecretMissing, got %v", err)
		}
	})

	t.Run("successful introspection active", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/introspect" {
				t.Errorf("expected /introspect, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"active":true}`)
		}))
		defer server.Close()

		auth := &oauthAuth{
			issuer:         server.URL,
			resourceSecret: "secret",
			httpClient:     server.Client(),
		}
		active, err := auth.validateToken(context.Background(), "valid-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !active {
			t.Error("expected active token")
		}
	})

	t.Run("introspection inactive", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"active":false}`)
		}))
		defer server.Close()

		auth := &oauthAuth{
			issuer:         server.URL,
			resourceSecret: "secret",
			httpClient:     server.Client(),
		}
		active, err := auth.validateToken(context.Background(), "expired-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active {
			t.Error("expected inactive token")
		}
	})

	t.Run("introspection server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		auth := &oauthAuth{
			issuer:         server.URL,
			resourceSecret: "secret",
			httpClient:     server.Client(),
		}
		_, err := auth.validateToken(context.Background(), "token")
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})
}

func TestAuthorizeRequest(t *testing.T) {
	t.Run("no oauth configured", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if !transport.authorizeRequest(w, req) {
			t.Error("expected authorized when oauth is nil")
		}
	})

	t.Run("missing bearer token", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		transport.oauth = &oauthAuth{issuer: "http://test", resourceSecret: "s", httpClient: &http.Client{}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Host = "localhost:8081"
		if transport.authorizeRequest(w, req) {
			t.Error("expected unauthorized without bearer token")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("empty bearer token", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		transport.oauth = &oauthAuth{issuer: "http://test", resourceSecret: "s", httpClient: &http.Client{}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Host = "localhost:8081"
		req.Header.Set("Authorization", "Bearer ")
		if transport.authorizeRequest(w, req) {
			t.Error("expected unauthorized for empty bearer token")
		}
	})
}

func TestHandleProtectedResourceMetadata(t *testing.T) {
	t.Run("no oauth returns 404", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
		setLocalhostHeaders(req)
		transport.handleProtectedResourceMetadata(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns metadata when oauth configured", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		transport.oauth = &oauthAuth{issuer: "http://issuer.test", resourceSecret: "s", httpClient: &http.Client{}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
		setLocalhostHeaders(req)
		transport.handleProtectedResourceMetadata(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var metadata protectedResourceMetadata
		if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !strings.Contains(metadata.Resource, "/mcp") {
			t.Errorf("expected resource to contain /mcp, got %q", metadata.Resource)
		}
		if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != "http://issuer.test" {
			t.Errorf("unexpected authorization_servers: %v", metadata.AuthorizationServers)
		}
	})

	t.Run("rejects POST method", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		transport.oauth = &oauthAuth{issuer: "http://issuer.test", resourceSecret: "s", httpClient: &http.Client{}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-protected-resource", nil)
		setLocalhostHeaders(req)
		transport.handleProtectedResourceMetadata(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})
}

func TestCompletionHandler(t *testing.T) {
	result, err := completionHandler(context.Background(), &mcp.CompleteRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Completion.Values) != 0 {
		t.Errorf("expected empty values, got %v", result.Completion.Values)
	}
}

func TestResourceSubscribeHandler(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("nil params", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{}); err == nil {
			t.Fatal("expected error for nil params")
		}
	})

	t.Run("empty URI", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{
			Params: &mcp.SubscribeParams{URI: ""},
		}); err == nil {
			t.Fatal("expected error for empty URI")
		}
	})

	t.Run("valid URI", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{
			Params: &mcp.SubscribeParams{URI: "campaigns://list"},
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceUnsubscribeHandler(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if err := resourceUnsubscribeHandler(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("valid URI", func(t *testing.T) {
		if err := resourceUnsubscribeHandler(context.Background(), &mcp.UnsubscribeRequest{
			Params: &mcp.UnsubscribeParams{URI: "campaigns://list"},
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGrpcAddress(t *testing.T) {
	tests := []struct {
		name     string
		fallback string
		want     string
	}{
		{"uses fallback when provided", "localhost:50051", "localhost:50051"},
		{"uses fallback for whitespace", "  ", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpcAddress(tt.fallback); got != tt.want {
				t.Errorf("grpcAddress(%q) = %q, want %q", tt.fallback, got, tt.want)
			}
		})
	}
}

func TestServerContext(t *testing.T) {
	s := &Server{}

	t.Run("default context is empty", func(t *testing.T) {
		ctx := s.getContext()
		if ctx.CampaignID != "" || ctx.SessionID != "" || ctx.ParticipantID != "" {
			t.Errorf("expected empty context, got %+v", ctx)
		}
	})

	t.Run("set and get context", func(t *testing.T) {
		s.setContext(domain.Context{CampaignID: "c1", SessionID: "s1"})
		ctx := s.getContext()
		if ctx.CampaignID != "c1" {
			t.Errorf("expected campaign_id %q, got %q", "c1", ctx.CampaignID)
		}
		if ctx.SessionID != "s1" {
			t.Errorf("expected session_id %q, got %q", "s1", ctx.SessionID)
		}
	})

	t.Run("nil server is safe", func(t *testing.T) {
		var nilServer *Server
		nilServer.setContext(domain.Context{CampaignID: "x"})
		ctx := nilServer.getContext()
		if ctx.CampaignID != "" {
			t.Errorf("expected empty context from nil server, got %+v", ctx)
		}
	})
}

func TestServerClose(t *testing.T) {
	t.Run("nil server is safe", func(t *testing.T) {
		var s *Server
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil conn is safe", func(t *testing.T) {
		s := &Server{}
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLoadOAuthAuthFromEnv(t *testing.T) {
	t.Run("empty issuer returns nil", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_MCP_OAUTH_ISSUER", "")
		t.Setenv("FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET", "")
		auth := loadOAuthAuthFromEnv()
		if auth != nil {
			t.Error("expected nil when issuer is empty")
		}
	})

	t.Run("whitespace issuer returns nil", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_MCP_OAUTH_ISSUER", "  ")
		auth := loadOAuthAuthFromEnv()
		if auth != nil {
			t.Error("expected nil for whitespace issuer")
		}
	})

	t.Run("configured issuer returns auth", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_MCP_OAUTH_ISSUER", "http://issuer.test/")
		t.Setenv("FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET", "my-secret")
		auth := loadOAuthAuthFromEnv()
		if auth == nil {
			t.Fatal("expected non-nil auth")
		}
		if auth.issuer != "http://issuer.test" {
			t.Errorf("expected trailing slash trimmed, got %q", auth.issuer)
		}
		if auth.resourceSecret != "my-secret" {
			t.Errorf("expected resource secret %q, got %q", "my-secret", auth.resourceSecret)
		}
		if auth.httpClient == nil {
			t.Error("expected non-nil http client")
		}
	})
}

func TestAuthorizeRequestWithValidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"active":true}`)
	}))
	defer server.Close()

	transport := NewHTTPTransport("localhost:8081")
	transport.oauth = &oauthAuth{
		issuer:         server.URL,
		resourceSecret: "secret",
		httpClient:     server.Client(),
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "localhost:8081"
	req.Header.Set("Authorization", "Bearer valid-token")
	if !transport.authorizeRequest(w, req) {
		t.Error("expected authorized for valid token")
	}
}

func TestAuthorizeRequestWithInactiveToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"active":false}`)
	}))
	defer server.Close()

	transport := NewHTTPTransport("localhost:8081")
	transport.oauth = &oauthAuth{
		issuer:         server.URL,
		resourceSecret: "secret",
		httpClient:     server.Client(),
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "localhost:8081"
	req.Header.Set("Authorization", "Bearer expired-token")
	if transport.authorizeRequest(w, req) {
		t.Error("expected unauthorized for inactive token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthorizeRequestWithMissingResourceSecret(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	transport.oauth = &oauthAuth{
		issuer:     "http://issuer.test",
		httpClient: &http.Client{},
		// resourceSecret deliberately empty
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "localhost:8081"
	req.Header.Set("Authorization", "Bearer some-token")
	if transport.authorizeRequest(w, req) {
		t.Error("expected unauthorized when resource secret is missing")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleHealth_POST(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	req := httptest.NewRequest(http.MethodPost, "/mcp/health", nil)
	setLocalhostHeaders(req)
	w := httptest.NewRecorder()
	transport.handleHealth(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHTTPConnectionWriteResponseRouting(t *testing.T) {
	ctx := context.Background()
	conn := &httpConnection{
		sessionID:   "test_session",
		reqChan:     make(chan jsonrpc.Message, 1),
		respChan:    make(chan jsonrpc.Message, 1),
		notifyChan:  make(chan jsonrpc.Message, 1),
		closed:      make(chan struct{}),
		ready:       make(chan struct{}, 1),
		pendingReqs: make(map[jsonrpc.ID]chan jsonrpc.Message),
	}

	// Register a pending request
	reqID, err := jsonrpc.MakeID("req-1")
	if err != nil {
		t.Fatalf("MakeID: %v", err)
	}
	respChan := make(chan jsonrpc.Message, 1)
	conn.pendingMu.Lock()
	conn.pendingReqs[reqID] = respChan
	conn.pendingMu.Unlock()

	// Write a response matching the pending request
	resp := &jsonrpc.Response{
		ID: reqID,
	}
	if err := conn.Write(ctx, resp); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// The response should be routed to the pending channel
	select {
	case msg := <-respChan:
		if msg == nil {
			t.Error("expected non-nil message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response on pending channel")
	}
}

func TestHTTPConnectionWriteNotification(t *testing.T) {
	ctx := context.Background()
	conn := &httpConnection{
		sessionID:   "test_session",
		reqChan:     make(chan jsonrpc.Message, 1),
		respChan:    make(chan jsonrpc.Message, 1),
		notifyChan:  make(chan jsonrpc.Message, 1),
		closed:      make(chan struct{}),
		ready:       make(chan struct{}, 1),
		pendingReqs: make(map[jsonrpc.ID]chan jsonrpc.Message),
	}

	// Write a notification (request without matching pending ID)
	notification := &jsonrpc.Request{
		Method: "notifications/resources/updated",
	}
	if err := conn.Write(ctx, notification); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// The notification should go to notifyChan
	select {
	case msg := <-conn.notifyChan:
		if msg == nil {
			t.Error("expected non-nil message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification")
	}
}

func TestHTTPConnectionReadClosed(t *testing.T) {
	conn := &httpConnection{
		sessionID:   "test_session",
		reqChan:     make(chan jsonrpc.Message, 1),
		respChan:    make(chan jsonrpc.Message, 1),
		notifyChan:  make(chan jsonrpc.Message, 1),
		closed:      make(chan struct{}),
		ready:       make(chan struct{}, 1),
		pendingReqs: make(map[jsonrpc.ID]chan jsonrpc.Message),
	}

	// Close the connection first
	conn.Close()

	// Read from closed connection should error
	_, err := conn.Read(context.Background())
	if err == nil {
		t.Fatal("expected error reading from closed connection")
	}
}

func TestHTTPConnectionReadContextCancelled(t *testing.T) {
	conn := &httpConnection{
		sessionID:   "test_session",
		reqChan:     make(chan jsonrpc.Message, 1),
		respChan:    make(chan jsonrpc.Message, 1),
		notifyChan:  make(chan jsonrpc.Message, 1),
		closed:      make(chan struct{}),
		ready:       make(chan struct{}, 1),
		pendingReqs: make(map[jsonrpc.ID]chan jsonrpc.Message),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRegisterToolsNoPanic(t *testing.T) {
	// Verify all registration functions can be called without panic
	// when given a real MCP server and nil gRPC clients.
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	server := &Server{}

	t.Run("registerDaggerheartTools", func(t *testing.T) {
		registerDaggerheartTools(mcpServer, nil)
	})

	t.Run("registerCampaignTools", func(t *testing.T) {
		registerCampaignTools(mcpServer, nil, nil, nil, nil, server.getContext, nil)
	})

	t.Run("registerSessionTools", func(t *testing.T) {
		registerSessionTools(mcpServer, nil, server.getContext, nil)
	})

	t.Run("registerEventTools", func(t *testing.T) {
		registerEventTools(mcpServer, nil, server.getContext)
	})

	t.Run("registerForkTools", func(t *testing.T) {
		registerForkTools(mcpServer, nil, nil)
	})

	t.Run("registerContextTools", func(t *testing.T) {
		registerContextTools(mcpServer, nil, nil, nil, server, nil)
	})

	t.Run("registerCampaignResources", func(t *testing.T) {
		registerCampaignResources(mcpServer, nil, nil, nil)
	})

	t.Run("registerSessionResources", func(t *testing.T) {
		registerSessionResources(mcpServer, nil)
	})

	t.Run("registerEventResources", func(t *testing.T) {
		registerEventResources(mcpServer, nil)
	})

	t.Run("registerContextResources", func(t *testing.T) {
		registerContextResources(mcpServer, server)
	})
}

func TestGrpcAddressFromEnv(t *testing.T) {
	t.Run("reads from env when fallback is empty", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "custom:9999")
		got := grpcAddress("")
		if got != "custom:9999" {
			t.Errorf("expected %q, got %q", "custom:9999", got)
		}
	})

	t.Run("fallback takes precedence over env", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "custom:9999")
		got := grpcAddress("default:50051")
		if got != "default:50051" {
			t.Errorf("expected %q, got %q", "default:50051", got)
		}
	})

	t.Run("empty fallback and empty env returns fallback", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "")
		got := grpcAddress("")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

func TestNewHTTPTransportDefaults(t *testing.T) {
	t.Run("empty addr defaults to localhost", func(t *testing.T) {
		transport := NewHTTPTransport("")
		if transport.addr != "localhost:8081" {
			t.Errorf("expected default addr %q, got %q", "localhost:8081", transport.addr)
		}
	})

	t.Run("sessions map initialized", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if transport.sessions == nil {
			t.Error("expected sessions map to be initialized")
		}
	})

	t.Run("serverOnce map initialized", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if transport.serverOnce == nil {
			t.Error("expected serverOnce map to be initialized")
		}
	})

	t.Run("serverCtx initialized", func(t *testing.T) {
		transport := NewHTTPTransport("localhost:8081")
		if transport.serverCtx == nil {
			t.Error("expected serverCtx to be initialized")
		}
		if transport.serverCancel == nil {
			t.Error("expected serverCancel to be initialized")
		}
	})
}

func TestBaseURLFromRequestTLS(t *testing.T) {
	t.Run("TLS connection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://secure.example.com/path", nil)
		req.Host = "secure.example.com"
		req.TLS = &tls.ConnectionState{}
		got := baseURLFromRequest(req)
		if got != "https://secure.example.com" {
			t.Errorf("expected %q, got %q", "https://secure.example.com", got)
		}
	})

	t.Run("X-Forwarded-Proto overrides TLS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
		req.Host = "example.com"
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("X-Forwarded-Proto", "http")
		got := baseURLFromRequest(req)
		if got != "http://example.com" {
			t.Errorf("expected %q, got %q", "http://example.com", got)
		}
	})
}

func TestHandleSSEWithSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	transport := NewHTTPTransport("localhost:8081")

	// Create a session
	conn, err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	sessionID := conn.SessionID()

	// Make SSE request with the valid session header
	req := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
	setLocalhostHeaders(req)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// handleSSE blocks until context is cancelled, so run in goroutine
	done := make(chan struct{})
	go func() {
		transport.handleSSE(w, req)
		close(done)
	}()

	// Cancel context to unblock SSE
	cancel()

	select {
	case <-done:
		// SSE handler returned
	case <-time.After(2 * time.Second):
		t.Fatal("handleSSE did not return after context cancellation")
	}

	// Check that SSE headers were set
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
}

func TestHandleSSEInvalidSessionHeader(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")

	req := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
	setLocalhostHeaders(req)
	req.Header.Set("Mcp-Session-Id", "nonexistent-session")
	w := httptest.NewRecorder()

	transport.handleSSE(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
