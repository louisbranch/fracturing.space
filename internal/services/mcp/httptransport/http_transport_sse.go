package httptransport

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// handleSSE handles GET /mcp for Server-Sent Events streaming.
// SSE is intentionally kept as a notification-only path so request/reply
// operations can be decoupled from streaming delivery and shared connection
// state can be updated in-place without response blocking.
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// SSE requires an existing bridge session ID.
	var session *httpSession
	var exists bool
	sessionID := r.Header.Get("Mcp-Session-Id")
	t.sessionsMu.RLock()
	session, exists = t.sessions[sessionID]
	t.sessionsMu.RUnlock()

	if !exists || session == nil {
		http.Error(w, "Invalid or missing session ID", http.StatusBadRequest)
		return
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
		case msg, ok := <-session.conn.notifyChan:
			if !ok || msg == nil {
				return
			}
			// Update lastUsed on each message
			t.sessionsMu.Lock()
			if s, ok := t.sessions[sessionID]; ok && s != nil {
				s.lastUsed = time.Now()
			}
			t.sessionsMu.Unlock()

			// Send as SSE event
			data, err := jsonrpc.EncodeMessage(msg)
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
