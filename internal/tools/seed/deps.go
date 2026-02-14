package seed

import "time"

// mcpClient abstracts the MCP stdio transport so tests can inject fakes.
type mcpClient interface {
	WriteMessage(message any) error
	ReadResponseForID(requestID any, timeout time.Duration) (any, []byte, error)
	Close()
}
