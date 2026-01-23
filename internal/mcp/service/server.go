package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	dualityv1 "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
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

// New creates a configured MCP server that connects to Duality and Campaign gRPC services.
func New(grpcAddr string) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	addr := grpcAddress(grpcAddr)
	conn, err := newGRPCConn(addr)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC server at %s: %w", addr, err)
	}

	dualityClient := dualityv1.NewDualityServiceClient(conn)
	campaignClient := campaignpb.NewCampaignServiceClient(conn)
	registerDualityTools(mcpServer, dualityClient)
	registerCampaignTools(mcpServer, campaignClient)

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
	if err := mcpServer.waitForHealth(ctx); err != nil {
		closeErr := mcpServer.Close()
		if closeErr != nil {
			return fmt.Errorf("wait for gRPC health: %v; close gRPC connection: %w", err, closeErr)
		}
		return err
	}
	return mcpServer.serveWithTransport(ctx, transport)
}

// newGRPCConn connects to the gRPC server shared by MCP services.
func newGRPCConn(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// grpcAddress resolves the gRPC address from env or defaults.
func grpcAddress(fallback string) string {
	if value := os.Getenv("DUALITY_GRPC_ADDR"); value != "" {
		return value
	}
	return fallback
}

func (s *Server) waitForHealth(ctx context.Context) error {
	if s == nil || s.conn == nil {
		return fmt.Errorf("gRPC connection is not configured")
	}

	healthClient := grpc_health_v1.NewHealthClient(s.conn)
	backoff := 200 * time.Millisecond
	for {
		callCtx, cancel := context.WithTimeout(ctx, time.Second)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		cancel()
		if err == nil && response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			log.Printf("gRPC health check is SERVING")
			return nil
		}
		if err != nil {
			log.Printf("waiting for gRPC health: %v", err)
		} else {
			log.Printf("waiting for gRPC health: status %s", response.GetStatus().String())
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for gRPC health: %w", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < time.Second {
			backoff *= 2
			if backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}
