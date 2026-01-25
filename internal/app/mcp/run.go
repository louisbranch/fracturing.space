package mcp

import (
	"context"
	"fmt"

	"github.com/louisbranch/duality-engine/internal/mcp/service"
)

// Run starts the MCP app with the provided gRPC address, HTTP address, and transport type.
func Run(ctx context.Context, grpcAddr, httpAddr, transport string) error {
	var transportKind service.TransportKind
	switch transport {
	case "http":
		transportKind = service.TransportHTTP
	case "stdio", "":
		transportKind = service.TransportStdio
	default:
		return fmt.Errorf("invalid transport %q: must be 'stdio' or 'http'", transport)
	}
	
	return service.Run(ctx, service.Config{
		GRPCAddr:  grpcAddr,
		HTTPAddr:  httpAddr,
		Transport: transportKind,
	})
}
