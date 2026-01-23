package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// serverName identifies this MCP server to clients.
	serverName = "Duality Engine MCP"
	// serverVersion identifies the MCP server version.
	serverVersion = "0.1.0"
)

// TransportKind identifies the MCP transport implementation.
type TransportKind string

const (
	// TransportStdio uses standard input/output for MCP.
	TransportStdio TransportKind = "stdio"
	// TransportHTTP is reserved for future HTTP transport support.
	TransportHTTP TransportKind = "http"
)

// Config configures the MCP server.
type Config struct {
	GRPCAddr  string
	Transport TransportKind
}

// Server hosts the MCP server.
type Server struct {
	mcpServer *mcp.Server
	conn      *grpc.ClientConn
}

// New creates a configured MCP server that connects to the gRPC dice service.
func New(grpcAddr string) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	addr := grpcAddress(grpcAddr)
	conn, grpcClient, err := newDualityClient(addr)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC server at %s: %w", addr, err)
	}

	registerDualityTools(mcpServer, grpcClient)

	return &Server{mcpServer: mcpServer, conn: conn}, nil
}

// Run creates and serves the MCP server until the context ends.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Transport == "" {
		cfg.Transport = TransportStdio
	}

	switch cfg.Transport {
	case TransportStdio:
		return runWithTransport(ctx, cfg.GRPCAddr, &mcp.StdioTransport{})
	case TransportHTTP:
		return fmt.Errorf("transport %q is not supported", cfg.Transport)
	default:
		return fmt.Errorf("transport %q is not supported", cfg.Transport)
	}
}

// Serve starts the MCP server on stdio and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	return s.serveWithTransport(ctx, &mcp.StdioTransport{})
}

// Close releases the gRPC connection held by the server.
func (s *Server) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	if err := s.conn.Close(); err != nil {
		return err
	}
	s.conn = nil
	return nil
}

// serveWithTransport starts the MCP server using the provided transport.
func (s *Server) serveWithTransport(ctx context.Context, transport mcp.Transport) error {
	if s == nil || s.mcpServer == nil {
		return fmt.Errorf("MCP server is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	err := s.mcpServer.Run(ctx, transport)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}
	closeErr := s.Close()
	if closeErr != nil {
		if err == nil {
			return fmt.Errorf("close gRPC connection: %w", closeErr)
		}
		return fmt.Errorf("serve MCP: %v; close gRPC connection: %w", err, closeErr)
	}
	if err != nil {
		return fmt.Errorf("serve MCP: %w", err)
	}
	return nil
}

// runWithTransport creates a server and serves it over the provided transport.
func runWithTransport(ctx context.Context, grpcAddr string, transport mcp.Transport) error {
	mcpServer, err := New(grpcAddr)
	if err != nil {
		return err
	}
	return mcpServer.serveWithTransport(ctx, transport)
}

// newDualityClient connects to the gRPC Duality service.
func newDualityClient(addr string) (*grpc.ClientConn, dualityv1.DualityServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return conn, dualityv1.NewDualityServiceClient(conn), nil
}

// grpcAddress resolves the gRPC address from env or defaults.
func grpcAddress(fallback string) string {
	if value := os.Getenv("DUALITY_GRPC_ADDR"); value != "" {
		return value
	}
	return fallback
}
