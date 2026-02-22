package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// httpConnection implements mcp.Connection for HTTP-based communication.
// The MCP SDK expects a bidirectional connection model, so this adapter maps
// request/response flow and notification delivery onto separate buffered channels.
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
// The split channel model avoids delivering unrelated notifications to callers that
// are awaiting a specific request/response exchange.
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
// Close is explicit about draining all waiters and channels so a dropped session
// cannot leave goroutines blocked on per-session reads/writes.
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
