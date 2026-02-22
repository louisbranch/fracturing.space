package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Connect implements mcp.Transport.Connect.
// For HTTP transport, this creates a new session and connection that will
// be used by the MCP server's Run method. The connection waits for HTTP requests.
// Each call creates a fresh session so one client identity can be tracked across
// multiple request/notification streams without cross-session contamination.
func (t *HTTPTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	sessionID := t.generateSessionID()

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

func (t *HTTPTransport) generateSessionID() string {
	randomReader := rand.Read
	if t != nil && t.randomReader != nil {
		randomReader = t.randomReader
	}
	return generateSessionIDWithRandomRead(randomReader)
}
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
	case <-t.readyAfterOrDefault()(t.serverReadyTimeoutOrDefault()):
		// Short timeout - if readiness hasn't happened yet, it will happen when
		// the first message is sent and Read() is called. This avoids blocking
		// tests that call ensureServerRunning() without sending messages.
	case <-t.serverCtx.Done():
		// Server is shutting down
		return
	}
}

func (t *HTTPTransport) readyAfterOrDefault() func(time.Duration) <-chan time.Time {
	if t == nil || t.readyAfter == nil {
		return time.After
	}
	return t.readyAfter
}

func (t *HTTPTransport) serverReadyTimeoutOrDefault() time.Duration {
	if t == nil || t.serverReadyTimeout <= 0 {
		return defaultSessionReadyTimeout
	}
	return t.serverReadyTimeout
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
	return generateSessionIDWithRandomRead(rand.Read)
}

func generateSessionIDWithRandomRead(randomRead func([]byte) (int, error)) string {
	// Generate 8 random bytes
	b := make([]byte, 8)
	if randomRead == nil {
		randomRead = rand.Read
	}
	if _, err := randomRead(b); err != nil {
		// Fallback to timestamp + counter if crypto/rand fails
		counter := sessionCounter.Add(1)
		return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), counter)
	}
	// Combine random bytes with counter for uniqueness
	counter := sessionCounter.Add(1)
	return fmt.Sprintf("session_%s_%d", hex.EncodeToString(b), counter)
}
