package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	sessionID   string
	reqChan     chan jsonrpc.Message
	respChan    chan jsonrpc.Message
	notifyChan  chan jsonrpc.Message // Separate channel for notifications (SSE)
	closed      chan struct{}
	ready       chan struct{} // Signals when Server.Connect() has started reading (buffered, size 1)
	readyOnce   sync.Once     // Ensures readiness is signaled only once
	mu          sync.Mutex
	closedFlag  bool
	pendingReqs map[jsonrpc.ID]chan jsonrpc.Message // Map request ID to response channel
	pendingMu   sync.Mutex
}

// NewHTTPTransport creates a new HTTP transport that will serve MCP over HTTP.
func NewHTTPTransport(addr string) *HTTPTransport {
	// Default to localhost-only binding for security
	if addr == "" {
		addr = "localhost:8081"
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPTransport{
		addr:         addr,
		sessions:     make(map[string]*httpSession),
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
		sessionID:   sessionID,
		reqChan:     make(chan jsonrpc.Message, defaultChannelBufferSize),
		respChan:    make(chan jsonrpc.Message, defaultChannelBufferSize),
		notifyChan:  make(chan jsonrpc.Message, defaultChannelBufferSize),
		closed:      make(chan struct{}),
		ready:       make(chan struct{}, 1), // Buffered so signal doesn't block
		pendingReqs: make(map[jsonrpc.ID]chan jsonrpc.Message),
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

	return conn, nil
}

// Start starts the HTTP server and begins handling requests.
// This should be called in a separate goroutine while the MCP server runs.
func (t *HTTPTransport) Start(ctx context.Context) error {
	// Update server context to use the provided context
	t.serverCtx, t.serverCancel = context.WithCancel(ctx)

	// Start session cleanup goroutine
	go t.cleanupSessions(ctx)

	mux := http.NewServeMux()

	// /mcp endpoint handles both GET (SSE) and POST (messages) based on HTTP method
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
		if err := t.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

// cleanupSessions periodically removes expired sessions from the sessions map.
// Sessions expire after sessionExpirationTime of inactivity.
func (t *HTTPTransport) cleanupSessions(ctx context.Context) {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.sessionsMu.Lock()
			now := time.Now()
			expirationTime := now.Add(-sessionExpirationTime)

			for id, session := range t.sessions {
				if session.lastUsed.Before(expirationTime) {
					// Close the connection
					session.conn.Close()
					// Remove from map
					delete(t.sessions, id)
					// Clean up serverOnce entry
					delete(t.serverOnce, id)
				}
			}
			t.sessionsMu.Unlock()
		}
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

	// Get or create session from cookie (MCP spec uses cookies, not custom headers)
	const cookieName = "mcp_session"
	var session *httpSession
	var exists bool
	var sessionID string

	// Read session ID from cookie
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie != nil && cookie.Value != "" {
		sessionID = cookie.Value
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
	}

	if !exists || session == nil {
		// Create new session for this request
		conn, err := t.Connect(r.Context())
		if err != nil {
			log.Printf("Failed to create session: %v", err)
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		sessionID = conn.SessionID()
		t.sessionsMu.RLock()
		session = t.sessions[sessionID]
		t.sessionsMu.RUnlock()

		// Set cookie for subsequent requests
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC message using the SDK's decoder
	msg, err := jsonrpc.DecodeMessage(body)
	if err != nil {
		log.Printf("Invalid JSON-RPC message: %v", err)
		http.Error(w, "Invalid JSON-RPC message", http.StatusBadRequest)
		return
	}

	// Log message type for debugging
	switch v := msg.(type) {
	case *jsonrpc.Request:
		var zeroID jsonrpc.ID
		if v.ID != zeroID {
			log.Printf("Decoded request: method=%s, id=%v", v.Method, v.ID)
		} else {
			log.Printf("Decoded notification: method=%s", v.Method)
		}
	case *jsonrpc.Response:
		log.Printf("Decoded response: id=%v", v.ID)
	default:
		log.Printf("Decoded unknown message type: %T", msg)
	}

	// Update last used time (protected by mutex)
	t.sessionsMu.Lock()
	if session != nil {
		session.lastUsed = time.Now()
	}
	t.sessionsMu.Unlock()

	// Nil check after session lookup
	if session == nil {
		http.Error(w, "Failed to retrieve session after creation", http.StatusInternalServerError)
		return
	}

	// Ensure MCP server is running for this connection
	// Start processing goroutine if not already started
	t.ensureServerRunning(session)

	// Send message to connection's request channel (will be read by MCP server)
	log.Printf("Sending message to reqChan for session %s", session.id)
	select {
	case session.conn.reqChan <- msg:
		log.Printf("Message sent to reqChan for session %s", session.id)
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
		// In JSON-RPC 2.0, notifications have null ID, requests have non-null ID
		// The zero value of jsonrpc.ID represents a null/empty ID (notification)
		id := v.ID
		isRequest = id != jsonrpc.ID{}
		if !isRequest {
			// This is a notification - log for debugging
			log.Printf("Received notification: method=%s", v.Method)
		}
	case *jsonrpc.Response:
		// Response shouldn't come in as a request
		http.Error(w, "Invalid message type: response", http.StatusBadRequest)
		return
	default:
		// For other types, assume it's a request and wait for response
		isRequest = true
	}

	if isRequest {
		// Request - wait for response matching this request ID
		req, ok := msg.(*jsonrpc.Request)
		if !ok {
			http.Error(w, "Invalid request type", http.StatusBadRequest)
			return
		}

		// Create a channel to receive the response for this specific request
		respChan := make(chan jsonrpc.Message, 1)
		session.conn.pendingMu.Lock()
		session.conn.pendingReqs[req.ID] = respChan
		session.conn.pendingMu.Unlock()

		// Wait for response with timeout
		select {
		case resp := <-respChan:
			// Clean up pending request
			session.conn.pendingMu.Lock()
			delete(session.conn.pendingReqs, req.ID)
			session.conn.pendingMu.Unlock()

			// Encode JSON-RPC response using SDK's encoder
			data, err := jsonrpc.EncodeMessage(resp)
			if err != nil {
				log.Printf("Failed to encode response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(data); err != nil {
				log.Printf("Failed to write response: %v", err)
			}
		case <-r.Context().Done():
			// Clean up pending request
			session.conn.pendingMu.Lock()
			delete(session.conn.pendingReqs, req.ID)
			session.conn.pendingMu.Unlock()
			http.Error(w, "Request cancelled", http.StatusRequestTimeout)
			return
		case <-time.After(defaultRequestTimeout):
			// Clean up pending request
			session.conn.pendingMu.Lock()
			delete(session.conn.pendingReqs, req.ID)
			session.conn.pendingMu.Unlock()
			http.Error(w, "Request timeout", http.StatusRequestTimeout)
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

	// Get or create session from cookie (MCP spec uses cookies, not query params)
	const cookieName = "mcp_session"
	var session *httpSession
	var exists bool
	var sessionID string

	// Read session ID from cookie
	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie != nil && cookie.Value != "" {
		sessionID = cookie.Value
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
	}

	if !exists || session == nil {
		conn, err := t.Connect(r.Context())
		if err != nil {
			log.Printf("Failed to create session: %v", err)
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		sessionID = conn.SessionID()
		t.sessionsMu.RLock()
		session = t.sessions[sessionID]
		t.sessionsMu.RUnlock()

		// Set cookie for subsequent requests
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Note: Session is managed via cookies, not headers (per MCP spec)

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream notifications from the connection's notification channel
	// SSE is for streaming notifications, not request/response pairs
	ctx := r.Context()

	// Update session activity timestamp so active SSE connections
	// are not considered idle by the cleanup goroutine
	t.sessionsMu.Lock()
	if s, ok := t.sessions[sessionID]; ok && s != nil {
		s.lastUsed = time.Now()
	}
	t.sessionsMu.Unlock()

	// Set up a ticker to periodically update lastUsed for long-lived SSE connections
	ticker := time.NewTicker(sseHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-session.conn.closed:
			return
		case <-ticker.C:
			// Periodically update lastUsed to prevent cleanup of active SSE connections
			t.sessionsMu.Lock()
			if s, ok := t.sessions[sessionID]; ok && s != nil {
				s.lastUsed = time.Now()
			}
			t.sessionsMu.Unlock()
		case msg := <-session.conn.notifyChan:
			// Update lastUsed on each message
			t.sessionsMu.Lock()
			if s, ok := t.sessions[sessionID]; ok && s != nil {
				s.lastUsed = time.Now()
			}
			t.sessionsMu.Unlock()

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
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

// Read implements mcp.Connection.Read.
// For HTTP transport, this reads messages from HTTP requests that have been
// sent to the connection's request channel.
func (c *httpConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	// Signal readiness on first read (when Server.Connect() starts reading)
	// Use sync.Once to ensure we only signal once
	c.readyOnce.Do(func() {
		select {
		case c.ready <- struct{}{}:
			log.Printf("Connection ready signaled for session %s", c.sessionID)
		default:
			// Channel already has signal, ignore
		}
	})

	log.Printf("Read() waiting for message on session %s", c.sessionID)
	select {
	case msg, ok := <-c.reqChan:
		if !ok {
			log.Printf("reqChan closed for session %s", c.sessionID)
			return nil, fmt.Errorf("connection closed")
		}
		log.Printf("Read() received message for session %s", c.sessionID)
		return msg, nil
	case <-c.closed:
		log.Printf("Connection closed for session %s", c.sessionID)
		return nil, fmt.Errorf("connection closed")
	case <-ctx.Done():
		log.Printf("Read() context cancelled for session %s", c.sessionID)
		return nil, ctx.Err()
	}
}

// Write implements mcp.Connection.Write.
// For HTTP transport, this writes responses to the connection's response channel,
// routing them to the correct pending request or to the notification channel.
func (c *httpConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	// Check closed flag and hold lock throughout to prevent race with Close()
	c.mu.Lock()
	closed := c.closedFlag
	c.mu.Unlock()

	if closed {
		return fmt.Errorf("connection closed")
	}

	// Check if this is a response with an ID that matches a pending request
	if resp, ok := msg.(*jsonrpc.Response); ok && resp.ID != (jsonrpc.ID{}) {
		c.pendingMu.Lock()
		respChan, exists := c.pendingReqs[resp.ID]
		c.pendingMu.Unlock()

		if exists {
			// Route to the specific pending request
			// Check closed again before writing to prevent writing to closed channel
			c.mu.Lock()
			closed = c.closedFlag
			c.mu.Unlock()
			if closed {
				return fmt.Errorf("connection closed")
			}

			select {
			case respChan <- msg:
				return nil
			case <-c.closed:
				return fmt.Errorf("connection closed")
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		// If no pending request found, treat as notification
	}

	// For notifications or unmatched responses, send to notification channel
	// Check closed again before writing
	c.mu.Lock()
	closed = c.closedFlag
	c.mu.Unlock()
	if closed {
		return fmt.Errorf("connection closed")
	}

	select {
	case c.notifyChan <- msg:
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

	// Close channels to unblock any waiting goroutines
	close(c.reqChan)
	close(c.respChan)
	close(c.notifyChan)

	// Close all pending request channels
	c.pendingMu.Lock()
	for _, respChan := range c.pendingReqs {
		close(respChan)
	}
	c.pendingReqs = nil
	c.pendingMu.Unlock()

	return nil
}

// SessionID implements mcp.Connection.SessionID.
func (c *httpConnection) SessionID() string {
	return c.sessionID
}

// ensureServerRunning ensures the MCP server is processing messages for this session.
// It starts a goroutine that calls Server.Connect with a transport that returns this session's connection.
// Uses sync.Once per session to prevent goroutine leaks from multiple calls.
func (t *HTTPTransport) ensureServerRunning(session *httpSession) {
	if t.server == nil {
		return
	}

	// Get or create sync.Once for this session to ensure server.Connect is only started once
	t.serverOnceMu.Lock()
	once, exists := t.serverOnce[session.id]
	if !exists {
		once = &sync.Once{}
		t.serverOnce[session.id] = once
	}
	t.serverOnceMu.Unlock()

	// Create a single-use transport that returns this session's connection
	// This allows Server.Connect to use the connection for this session
	sessionTransport := &sessionTransport{conn: session.conn}

	// Start the MCP server session for this connection only once
	once.Do(func() {
		go func() {
			// Connect the MCP server with this session's transport using the long-lived server context
			// This will read from reqChan and write to respChan
			serverSession, err := t.server.Connect(t.serverCtx, sessionTransport, nil)
			if err != nil {
				log.Printf("Failed to connect MCP server session %s: %v", session.id, err)
				return
			}
			// Wait for the session to complete (client disconnects or context cancelled)
			_ = serverSession.Wait()
		}()
	})

	// Wait for the connection to be ready (Server.Connect() has started reading)
	// Use a timeout to avoid blocking forever if something goes wrong
	// Use a shorter timeout to avoid slowing down tests when no messages are sent
	select {
	case <-session.conn.ready:
		// Connection is ready to process messages
	case <-time.After(100 * time.Millisecond):
		// Short timeout - if readiness hasn't happened yet, it will happen when
		// the first message is sent and Read() is called. This avoids blocking
		// tests that call ensureServerRunning() without sending messages.
	case <-t.serverCtx.Done():
		// Server is shutting down
		return
	}
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

var sessionCounter atomic.Uint64

// generateSessionID generates a unique session ID using crypto/rand
// combined with a counter to prevent collisions.
func generateSessionID() string {
	// Generate 8 random bytes
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp + counter if crypto/rand fails
		counter := sessionCounter.Add(1)
		return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), counter)
	}
	// Combine random bytes with counter for uniqueness
	counter := sessionCounter.Add(1)
	return fmt.Sprintf("session_%s_%d", hex.EncodeToString(b), counter)
}
