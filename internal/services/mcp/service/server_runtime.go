package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// completionHandler handles completion/complete requests with empty results.
// Returning empty completions is intentional today because MCP prompt/resource
// completion is still experimental in this codebase and would be unreliable
// without full context wiring.
// TODO: Return context-aware completions for prompt arguments and resource templates.
// That capability is intentionally deferred so early MCP clients stay predictable
// while protocol-level correctness is stabilized.
func completionHandler(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values: []string{},
		},
	}, nil
}

// resourceSubscribeHandler accepts resource subscriptions with a valid URI.
// The handler currently validates only addressing because MCP subscription
// semantics are delegated to the resource model once URI mapping is stable.
func resourceSubscribeHandler(_ context.Context, req *mcp.SubscribeRequest) error {
	if req == nil || req.Params == nil || strings.TrimSpace(req.Params.URI) == "" {
		return fmt.Errorf("resource uri is required")
	}
	return nil
}

// resourceUnsubscribeHandler accepts resource unsubscriptions with a valid URI.
// URI-level validation is still the boundary because MCP unsubscription is a
// resource routing signal, not yet a domain mutation path.
func resourceUnsubscribeHandler(_ context.Context, req *mcp.UnsubscribeRequest) error {
	if req == nil || req.Params == nil || strings.TrimSpace(req.Params.URI) == "" {
		return fmt.Errorf("resource uri is required")
	}
	return nil
}

// Run is the service entrypoint for MCP and blocks until context cancellation.
// It is intentionally transport-agnostic so startup can choose stdio for local
// tools and HTTP for browser/remote integrations.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Transport == "" {
		cfg.Transport = TransportStdio
	}

	switch cfg.Transport {
	case TransportStdio:
		return runWithTransport(ctx, cfg.GRPCAddr, &mcp.StdioTransport{})
	case TransportHTTP:
		return runWithHTTPTransport(ctx, cfg)
	default:
		return fmt.Errorf("transport %q is not supported", cfg.Transport)
	}
}

// runWithHTTPTransport creates a server and serves it over HTTP transport.
// runWithHTTPTransport keeps HTTP session/stateful transport concerns isolated from
// the same MCP domain handlers used by stdio.
func runWithHTTPTransport(ctx context.Context, cfg Config) error {
	// Default to localhost-only binding for security
	httpAddr := cfg.HTTPAddr
	if httpAddr == "" {
		httpAddr = "localhost:8081"
	}

	addr := grpcAddress(cfg.GRPCAddr)
	conn, err := dialGameGRPC(ctx, addr)
	if err != nil {
		return err
	}
	mcpServer, err := newServer(conn)
	if err != nil {
		return err
	}
	defer mcpServer.Close()

	// Start gRPC connection health monitoring in background
	// This ensures we detect connection failures during HTTP server operation
	healthCtx, healthCancel := context.WithCancel(ctx)
	defer healthCancel()
	go mcpServer.monitorHealth(healthCtx)

	// Create HTTP transport with reference to MCP server
	httpTransport := NewHTTPTransportWithServer(httpAddr, mcpServer.mcpServer)
	httpTransport.applyConfig(cfg)

	// Start HTTP server (this will handle all HTTP requests)
	return httpTransport.Start(ctx)
}

// monitorHealth periodically checks gRPC connection health.
// If the connection becomes unhealthy, it logs errors but doesn't terminate
// the HTTP server, allowing for graceful degradation while still surfacing
// connector issues quickly.
func (s *Server) monitorHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.conn == nil {
				log.Printf("gRPC connection is nil, health check skipped")
				continue
			}

			healthClient := grpc_health_v1.NewHealthClient(s.conn)
			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
			cancel()

			if err != nil {
				log.Printf("gRPC health check failed: %v", err)
			} else if response.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
				log.Printf("gRPC health check status: %s", response.GetStatus().String())
			}
			// Note: We log but don't fail - HTTP server continues to operate
			// Individual requests will handle gRPC errors appropriately
		}
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
// The server and its gRPC connection share a single exit path so cleanup behavior
// is consistent for both stdio and HTTP runs.
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

// setContext updates the server's context state.
func (s *Server) setContext(ctx domain.Context) {
	if s == nil {
		return
	}
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	s.ctx = ctx
}

// getContext returns the server's current context state.
func (s *Server) getContext() domain.Context {
	if s == nil {
		return domain.Context{}
	}
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	return s.ctx
}

// runWithTransport creates a server and serves it over the provided transport.
func runWithTransport(ctx context.Context, grpcAddr string, transport mcp.Transport) error {
	addr := grpcAddress(grpcAddr)
	conn, err := dialGameGRPC(ctx, addr)
	if err != nil {
		return err
	}
	mcpServer, err := newServer(conn)
	if err != nil {
		return err
	}
	return mcpServer.serveWithTransport(ctx, transport)
}

func dialGameGRPC(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}
	dialOpts := append(
		platformgrpc.DefaultClientDialOptions(),
		grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(mcpAuthzOverrideReason)),
		grpc.WithChainStreamInterceptor(grpcauthctx.AdminOverrideStreamClientInterceptor(mcpAuthzOverrideReason)),
	)
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		addr,
		timeouts.GRPCDial,
		logf,
		dialOpts...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageConnect {
				return nil, fmt.Errorf("connect to game server at %s: %w", addr, dialErr.Err)
			}
			return nil, dialErr.Err
		}
		return nil, err
	}
	return conn, nil
}

// newGRPCConn connects to the game server shared by MCP services.
func newGRPCConn(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(mcpAuthzOverrideReason)),
		grpc.WithChainStreamInterceptor(grpcauthctx.AdminOverrideStreamClientInterceptor(mcpAuthzOverrideReason)),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// grpcAddress resolves the gRPC address from the explicit fallback or env when empty.
func grpcAddress(fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	if value := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_ADDR")); value != "" {
		return value
	}
	return fallback
}

func (s *Server) waitForHealth(ctx context.Context) error {
	if s == nil || s.conn == nil {
		return fmt.Errorf("gRPC connection is not configured")
	}

	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}
	return platformgrpc.WaitForHealth(ctx, s.conn, "", logf)
}
