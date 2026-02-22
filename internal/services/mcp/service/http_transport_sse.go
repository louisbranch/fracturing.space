package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// handleSSE handles GET /mcp/sse for Server-Sent Events streaming.
// SSE is intentionally kept as a notification-only path so request/reply
// operations can be decoupled from streaming delivery and shared connection
// state can be updated in-place without response blocking.
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !t.authorizeRequest(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session from header or cookie
	const cookieName = "mcp_session"
	var session *httpSession
	var exists bool
	var sessionID string

	sessionID = strings.TrimSpace(r.Header.Get("Mcp-Session-Id"))
	if sessionID != "" {
		t.sessionsMu.RLock()
		session, exists = t.sessions[sessionID]
		t.sessionsMu.RUnlock()
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
		http.Error(w, "Invalid or missing session ID", http.StatusBadRequest)
		return
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
