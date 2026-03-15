package seed

import (
	"context"
	"time"
)

// mcpClient abstracts raw MCP request/response transport so tests can inject fakes.
type mcpClient interface {
	WriteMessage(message any) error
	ReadResponseForID(ctx context.Context, requestID any, timeout time.Duration) (any, []byte, error)
	Close()
}
