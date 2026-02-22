package service

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// handleMessages handles POST /mcp/messages for JSON-RPC requests.
// It maps transport-agnostic JSON-RPC payloads onto session-local MCP
// connection state so one campaign/auth participant can stay contiguous across
// multiple HTTP round-trips.
// It is the write path for all request/notification traffic and performs
// per-session validation before routing into the MCP runtime.
func (t *HTTPTransport) handleMessages(w http.ResponseWriter, r *http.Request) {
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !t.authorizeRequest(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
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

	// Determine if this is an initialization request.
	// The MCP HTTP transport requires initialize before other methods.
	isInitialize := false
	if req, ok := msg.(*jsonrpc.Request); ok {
		isInitialize = req.Method == "initialize"
	}

	// Get or create session from header or cookie
	const cookieName = "mcp_session"
	var session *httpSession
	var exists bool
	var sessionID string

	sessionID = strings.TrimSpace(r.Header.Get("Mcp-Session-Id"))
	if sessionID != "" {
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
		if !exists || session == nil {
			if !isInitialize {
				writeSessionError(w, "Invalid session ID")
				return
			}
			session = nil
			exists = false
			sessionID = ""
		}
	} else {
		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie != nil && cookie.Value != "" {
			sessionID = cookie.Value
			t.sessionsMu.RLock()
			session, exists = t.sessions[sessionID]
			t.sessionsMu.RUnlock()
		}
	}

	if !exists || session == nil {
		if !isInitialize {
			writeSessionError(w, "Invalid or missing session ID")
			return
		}
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

		// Set session header for MCP clients
		w.Header().Set("Mcp-Session-Id", sessionID)

		// Set cookie for subsequent requests (legacy fallback)
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
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

		// Send message to connection's request channel (will be read by MCP server)
		log.Printf("Sending message to reqChan for session %s", session.id)
		select {
		case session.conn.reqChan <- msg:
			log.Printf("Message sent to reqChan for session %s", session.id)
		case <-r.Context().Done():
			// Clean up pending request
			session.conn.pendingMu.Lock()
			delete(session.conn.pendingReqs, req.ID)
			session.conn.pendingMu.Unlock()
			http.Error(w, "Request cancelled", http.StatusRequestTimeout)
			return
		}

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
		// Send message to connection's request channel (will be read by MCP server)
		log.Printf("Sending message to reqChan for session %s", session.id)
		select {
		case session.conn.reqChan <- msg:
			log.Printf("Message sent to reqChan for session %s", session.id)
		case <-r.Context().Done():
			http.Error(w, "Request cancelled", http.StatusRequestTimeout)
			return
		}

		// Notification - no response
		w.WriteHeader(http.StatusNoContent)
	}
}
func writeSessionError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32000,
			"message": message,
		},
		"id": nil,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		_, _ = w.Write([]byte("{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32000,\"message\":\"Session error\"},\"id\":null}"))
		return
	}
	_, _ = w.Write(data)
}
