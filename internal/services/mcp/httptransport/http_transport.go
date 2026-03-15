package httptransport

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var listenTCP = net.Listen
var newTLSListener = tls.NewListener

// ErrSessionBootstrapRejected reports that the service runtime rejected an
// initialize request before an MCP session could be created.
var ErrSessionBootstrapRejected = errors.New("mcp session bootstrap rejected")

// mcpHTTPEnv holds env-parsed configuration for MCP HTTP transport.
type mcpHTTPEnv struct {
	AllowedHosts []string `env:"FRACTURING_SPACE_MCP_ALLOWED_HOSTS"          envSeparator:","`
}

const (
	// defaultChannelBufferSize is the buffer size for request, response, and notification channels.
	// This allows some buffering of messages before blocking, improving throughput under load.
	defaultChannelBufferSize = 10

	// defaultRequestTimeout is the maximum time to wait for a JSON-RPC response.
	// This should be long enough for most operations but short enough to fail fast on errors.
	defaultRequestTimeout = 30 * time.Second

	// defaultShutdownTimeout is the maximum time to wait for graceful HTTP server shutdown.
	// This should be longer than defaultRequestTimeout to allow in-flight requests to complete.
	defaultShutdownTimeout = 35 * time.Second

	// sessionCleanupInterval is how often the cleanup goroutine runs to remove expired sessions.
	sessionCleanupInterval = 5 * time.Minute

	// sessionExpirationTime is how long a session can be inactive before being cleaned up.
	sessionExpirationTime = 1 * time.Hour

	// sseHeartbeatInterval is how often to update lastUsed for active SSE connections.
	sseHeartbeatInterval = 30 * time.Second

	// defaultSessionReadyTimeout bounds how long we wait for a session connection
	// to become ready before request handling continues.
	defaultSessionReadyTimeout = 100 * time.Millisecond
)

// SessionRuntime exposes the per-session MCP runtime owned by the service
// package without leaking service internals into the HTTP transport package.
type SessionRuntime interface {
	// Server returns the MCP server bound to this session's fixed authority.
	Server() *mcp.Server
	// Close releases any service-owned resources associated with the session.
	Close() error
}

// RuntimeFactory exposes the base MCP server and optional per-session runtime
// creation that the HTTP bridge needs to keep session authority isolated.
type RuntimeFactory interface {
	// BaseServer returns the shared MCP runtime used when a session does not
	// need a dedicated authority binding.
	BaseServer() *mcp.Server
	// NewSessionRuntime creates a session-scoped runtime when bridge headers pin
	// the MCP session to one fixed internal AI authority.
	NewSessionRuntime(header http.Header) (SessionRuntime, error)
}

// HTTPTransport implements mcp.Transport for HTTP-based MCP communication.
// It provides an HTTP server that handles JSON-RPC messages over POST requests
// and supports Server-Sent Events (SSE) for streaming responses.
// The implementation is intentionally explicit about session lifecycle and cleanup so
// long-lived local MCP clients cannot leak resources even when upstream services
// stop responding.
type HTTPTransport struct {
	addr         string
	allowedHosts map[string]struct{}
	server       *mcp.Server
	runtime      RuntimeFactory
	sessions     map[string]*httpSession
	sessionsMu   sync.RWMutex
	httpServer   *http.Server
	serverCtx    context.Context
	serverCancel context.CancelFunc
	serverOnceMu sync.Mutex
	serverOnce   map[string]*sync.Once
	tlsConfig    *tls.Config

	serverReadyTimeout time.Duration
	randomReader       func([]byte) (int, error)
	readyAfter         func(time.Duration) <-chan time.Time
}

// SetTLSConfig records the TLS listener configuration used when the HTTP
// transport serves on an externally terminated port.
func (t *HTTPTransport) SetTLSConfig(cfg *tls.Config) {
	if t == nil {
		return
	}
	t.tlsConfig = cfg
}

// httpSession maintains state for a single MCP session in memory.
// It tracks liveness and the active connection so cleanup and SSE delivery can
// be scoped to one browser/process session.
type httpSession struct {
	id        string
	conn      *httpConnection
	runtime   SessionRuntime
	createdAt time.Time
	lastUsed  time.Time
}

// NewHTTPTransport creates a new HTTP transport that will serve MCP over HTTP.
// It defaults to localhost-only binding to keep the default footprint constrained
// to local development unless explicit host configuration broadens access.
func NewHTTPTransport(addr string) *HTTPTransport {
	// Default to localhost-only binding for security
	if addr == "" {
		addr = "localhost:8085"
	}
	var raw mcpHTTPEnv
	_ = config.ParseEnv(&raw)
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPTransport{
		addr:               addr,
		allowedHosts:       parseAllowedHosts(raw.AllowedHosts),
		sessions:           make(map[string]*httpSession),
		serverCtx:          ctx,
		serverCancel:       cancel,
		serverOnce:         make(map[string]*sync.Once),
		serverReadyTimeout: defaultSessionReadyTimeout,
		randomReader:       rand.Read,
		readyAfter:         time.After,
	}
}

// NewHTTPTransportWithServer creates a new HTTP transport with a reference to the MCP server.
//
// Callers use this when they need to inject a preconfigured MCP runtime without
// re-dialing transport setup, which keeps tests and process lifecycle simpler.
func NewHTTPTransportWithServer(addr string, server *mcp.Server) *HTTPTransport {
	transport := NewHTTPTransport(addr)
	transport.server = server
	return transport
}

// NewHTTPTransportWithRuntime creates a new HTTP transport with access to the
// MCP runtime factory so each HTTP session can bind dedicated authority when
// the internal AI bridge supplies fixed session headers.
func NewHTTPTransportWithRuntime(addr string, runtime RuntimeFactory) *HTTPTransport {
	transport := NewHTTPTransport(addr)
	transport.runtime = runtime
	if runtime != nil {
		transport.server = runtime.BaseServer()
	}
	return transport
}

// newSessionRuntime asks the owning service runtime for a dedicated session
// server when the bridge session binds fixed AI authority.
func (t *HTTPTransport) newSessionRuntime(header http.Header) (SessionRuntime, error) {
	if t == nil || t.runtime == nil {
		return nil, nil
	}
	return t.runtime.NewSessionRuntime(header)
}

// Start starts the HTTP server and begins handling requests.
// This should be called in a separate goroutine while the MCP server runs.
// The same server instance multiplexes POST requests and SSE streams while sharing
// host validation and session lifecycle enforcement.
func (t *HTTPTransport) Start(ctx context.Context) error {
	// Update server context to use the provided context
	t.serverCtx, t.serverCancel = context.WithCancel(ctx)

	// Start session cleanup goroutine
	go t.cleanupSessions(ctx)

	mux := http.NewServeMux()

	// /mcp endpoint handles both GET (SSE) and POST (messages) based on HTTP method.
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			t.handleSSE(w, r)
		case http.MethodPost:
			t.handleMessages(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /mcp/health - Health check endpoint
	mux.HandleFunc("/mcp/health", t.handleHealth)

	t.httpServer = &http.Server{
		Addr:    t.addr,
		Handler: mux,
	}

	log.Printf("Starting MCP HTTP server on %s", t.addr)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		listener, err := listenTCP("tcp", t.addr)
		if err != nil {
			errChan <- err
			return
		}

		serverListener := listener
		if t.tlsConfig != nil {
			serverListener = newTLSListener(listener, t.tlsConfig)
		}

		if err := t.httpServer.Serve(serverListener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutting down MCP HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		if err := t.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		// Cancel server context to stop all server.Run goroutines
		if t.serverCancel != nil {
			t.serverCancel()
		}
		return nil
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	}
}
