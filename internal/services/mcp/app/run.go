package mcp

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/mcp/service"
)

// Run starts the MCP app with the provided game/AI gRPC and HTTP addresses.
func Run(ctx context.Context, grpcAddr, aiAddr, httpAddr string, profile service.RegistrationProfile) error {
	return service.Run(ctx, service.Config{
		GRPCAddr:            grpcAddr,
		AIAddr:              aiAddr,
		HTTPAddr:            httpAddr,
		RegistrationProfile: profile,
	})
}
