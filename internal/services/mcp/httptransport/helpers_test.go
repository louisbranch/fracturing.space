package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeRuntimeFactory struct {
	baseServer        *mcp.Server
	newSessionRuntime func(http.Header) (SessionRuntime, error)
}

func (f fakeRuntimeFactory) BaseServer() *mcp.Server {
	return f.baseServer
}

func (f fakeRuntimeFactory) NewSessionRuntime(header http.Header) (SessionRuntime, error) {
	if f.newSessionRuntime == nil {
		return nil, nil
	}
	return f.newSessionRuntime(header)
}

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

	reqID, err := jsonrpc.MakeID("req-1")
	if err != nil {
		t.Fatalf("MakeID: %v", err)
	}
	respChan := make(chan jsonrpc.Message, 1)
	conn.pendingMu.Lock()
	conn.pendingReqs[reqID] = respChan
	conn.pendingMu.Unlock()

	resp := &jsonrpc.Response{ID: reqID}
	if err := conn.Write(ctx, resp); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

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

	notification := &jsonrpc.Request{Method: "notifications/resources/updated"}
	if err := conn.Write(ctx, notification); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

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

	_ = conn.Close()

	if _, err := conn.Read(context.Background()); err == nil {
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
	cancel()

	if _, err := conn.Read(ctx); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestNewHTTPTransportDefaults(t *testing.T) {
	t.Run("empty addr defaults to localhost", func(t *testing.T) {
		transport := NewHTTPTransport("")
		if transport.addr != "localhost:8085" {
			t.Errorf("expected default addr %q, got %q", "localhost:8085", transport.addr)
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

func TestHandleSSEWithSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	transport := NewHTTPTransport("localhost:8081")

	conn, err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	sessionID := conn.SessionID()

	req := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
	setLocalhostHeaders(req)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		transport.handleSSE(w, req)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleSSE did not return after context cancellation")
	}

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
