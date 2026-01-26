//go:build !conformance

package conformance

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register is a no-op unless the conformance build tag is enabled.
func Register(_ *mcp.Server) {}
