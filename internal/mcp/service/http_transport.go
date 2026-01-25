package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// HTTPTransport implements mcp.Transport for HTTP-based MCP communication.
// It provides an HTTP server that handles JSON-RPC messages over POST requests
// and supports Server-Sent Events (SSE) for streaming responses.
//
// TODO: Add authentication/authorization for production use
// TODO: Add rate limiting per connection
type HTTPTransport struct {
	addr         string
	server       *mcp.Server
	sessions     map[string]*httpSession
	sessionsMu   sync.RWMutex
	httpServer   *http.Server
	connChan     chan *httpConnection
	serverCtx    context.Context
	serverCancel context.CancelFunc
	serverOnceMu sync.Mutex
	serverOnce   map[string]*sync.Once
}

// httpSession maintains state for an HTTP client connection.
type httpSession struct {
	id        string
	conn      *httpConnection
	createdAt time.Time
	lastUsed  time.Time
}

// httpConnection implements mcp.Connection for HTTP-based communication.
type httpConnection struct {
	sessionID  string
	reqChan    chan jsonrpc.Message
	respChan   chan jsonrpc.Message
	closed     chan struct{}
	mu         sync.Mutex
	closedFlag bool
}

// NewHTTPTransport creates a new HTTP transport that will serve MCP over HTTP.
func NewHTTPTransport(addr string) *HTTPTransport {
	// TODO: Bind HTTP server to localhost only by default (not 0.0.0.0) for security
	if addr == "" {
		addr = "localhost:8081"
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPTransport{
		addr:         addr,
		sessions:     make(map[string]*httpSession),
		connChan:     make(chan *httpConnection, 10),
		serverCtx:    ctx,
		serverCancel: cancel,
		serverOnce:   make(map[string]*sync.Once),
	}
}

// NewHTTPTransportWithServer creates a new HTTP transport with a reference to the MCP server.
func NewHTTPTransportWithServer(addr string, server *mcp.Server) *HTTPTransport {
	transport := NewHTTPTransport(addr)
	transport.server = server
	return transport
}

// Connect implements mcp.Transport.Connect.
// For HTTP transport, this creates a new session and connection that will
// be used by the MCP server's Run method. The connection waits for HTTP requests.
func (t *HTTPTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	sessionID := generateSessionID()

	conn := &httpConnection{
		sessionID: sessionID,
		reqChan:   make(chan jsonrpc.Message, 10),
		respChan:  make(chan jsonrpc.Message, 10),
		closed:    make(chan struct{}),
	}

	session := &httpSession{
		id:        sessionID,
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}

	t.sessionsMu.Lock()
	t.sessions[sessionID] = session
	t.sessionsMu.Unlock()

	// Notify that a new connection is available
	select {
	case t.connChan <- conn:
	default:
	}

	return conn, nil
}

// Start starts the HTTP server and begins handling requests.
// This should be called in a separate goroutine while the MCP server runs.
func (t *HTTPTransport) Start(ctx context.Context) error {
	// Update server context to use the provided context
	t.serverCtx, t.serverCancel = context.WithCancel(ctx)

	mux := http.NewServeMux()

	// POST /mcp/messages - JSON-RPC request/response
	mux.HandleFunc("/mcp/messages", t.handleMessages)

	// GET /mcp/sse - Server-Sent Events stream
	mux.HandleFunc("/mcp/sse", t.handleSSE)

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
		if err := t.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutting down MCP HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// handleMessages handles POST /mcp/messages for JSON-RPC requests.
func (t *HTTPTransport) handleMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: Add API key/token authentication middleware
	// TODO: Add CORS headers if web clients are expected

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get or create session from header
	sessionID := r.Header.Get("X-MCP-Session-ID")
	var session *httpSession
	var exists bool

	if sessionID != "" {
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
	}

	if !exists || session == nil {
		// Create new session for this request
		conn, err := t.Connect(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
			return
		}
		sessionID = conn.SessionID()
		t.sessionsMu.RLock()
		session = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
		w.Header().Set("X-MCP-Session-ID", sessionID)
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC message
	var msg jsonrpc.Message
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON-RPC message: %v", err), http.StatusBadRequest)
		return
	}

	// Update last used time (protected by mutex)
	t.sessionsMu.Lock()
	session.lastUsed = time.Now()
	t.sessionsMu.Unlock()

	// Ensure MCP server is running for this connection
	// Start processing goroutine if not already started
	t.ensureServerRunning(session)

	// Send message to connection's request channel (will be read by MCP server)
	select {
	case session.conn.reqChan <- msg:
	case <-r.Context().Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	}

	// Check if message is a request (has ID) or notification (no ID)
	// Message is an interface, so we need to check the concrete type
	var isRequest bool
	switch v := msg.(type) {
	case *jsonrpc.Request:
		// Request has an ID field - check if it's set (not zero value)
		id := v.ID
		isRequest = id != jsonrpc.ID{}
	case *jsonrpc.Response:
		// Response shouldn't come in as a request
		http.Error(w, "Invalid message type: response", http.StatusBadRequest)
		return
	default:
		// For other types, assume it's a request and wait for response
		isRequest = true
	}

	if isRequest {
		// Request - wait for response
		select {
		case resp := <-session.conn.respChan:
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Printf("Failed to encode response: %v", err)
			}
		case <-r.Context().Done():
			http.Error(w, "Request cancelled", http.StatusRequestTimeout)
			return
		}
	} else {
		// Notification - no response
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleSSE handles GET /mcp/sse for Server-Sent Events streaming.
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// TODO: Add authentication check before establishing SSE connection

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get or create session
	sessionID := r.URL.Query().Get("session")
	var session *httpSession
	var exists bool

	if sessionID != "" {
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
	}

	if !exists || session == nil {
		conn, err := t.Connect(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
			return
		}
		sessionID = conn.SessionID()
		t.sessionsMu.RLock()
		session = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-MCP-Session-ID", sessionID)

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream messages from the connection's response channel
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-session.conn.closed:
			return
		case msg := <-session.conn.respChan:
			// Send as SSE event
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal SSE message: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// handleHealth handles GET /mcp/health for health checks.
func (t *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Read implements mcp.Connection.Read.
// For HTTP transport, this reads messages from HTTP requests that have been
// sent to the connection's request channel.
func (c *httpConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case msg := <-c.reqChan:
		return msg, nil
	case <-c.closed:
		return nil, fmt.Errorf("connection closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Write implements mcp.Connection.Write.
// For HTTP transport, this writes responses to the connection's response channel,
// which are then sent back to HTTP clients.
func (c *httpConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closedFlag {
		return fmt.Errorf("connection closed")
	}

	select {
	case c.respChan <- msg:
		return nil
	case <-c.closed:
		return fmt.Errorf("connection closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close implements mcp.Connection.Close.
func (c *httpConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closedFlag {
		return nil
	}

	c.closedFlag = true
	close(c.closed)
	return nil
}

// SessionID implements mcp.Connection.SessionID.
func (c *httpConnection) SessionID() string {
	return c.sessionID
}

// ensureServerRunning ensures the MCP server is processing messages for this session.
// It starts a goroutine that runs Server.Run with a transport that returns this session's connection.
// Uses sync.Once per session to prevent goroutine leaks from multiple calls.
func (t *HTTPTransport) ensureServerRunning(session *httpSession) {
	if t.server == nil {
		return
	}

	// Get or create sync.Once for this session to ensure server.Run is only started once
	t.serverOnceMu.Lock()
	once, exists := t.serverOnce[session.id]
	if !exists {
		once = &sync.Once{}
		t.serverOnce[session.id] = once
	}
	t.serverOnceMu.Unlock()

	// Create a single-use transport that returns this session's connection
	// This allows Server.Run to use the connection for this session
	sessionTransport := &sessionTransport{conn: session.conn}

	// Start the MCP server for this session only once
	once.Do(func() {
		go func() {
			// Run the MCP server with this session's transport using the long-lived server context
			// This will read from reqChan and write to respChan
			_ = t.server.Run(t.serverCtx, sessionTransport)
		}()
	})
}

// sessionTransport is a transport that returns a specific connection.
// This allows us to use Server.Run with a pre-existing connection.
type sessionTransport struct {
	conn mcp.Connection
}

// Connect implements mcp.Transport.Connect.
// It returns the pre-configured connection for this session.
func (st *sessionTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	return st.conn, nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
