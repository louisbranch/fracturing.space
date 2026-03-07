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

var dialGameRuntimeConn = dialGameGRPC

const (
	defaultHealthMonitorInterval = 30 * time.Second
	defaultHealthCheckTimeout    = 5 * time.Second
)

var (
	newHealthMonitorTicker = func(interval time.Duration) (<-chan time.Time, func()) {
		ticker := time.NewTicker(interval)
		return ticker.C, ticker.Stop
	}
	checkConnectionHealth = func(ctx context.Context, conn *grpc.ClientConn) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
		healthClient := grpc_health_v1.NewHealthClient(conn)
		response, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		if err != nil {
			return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
		}
		return response.GetStatus(), nil
	}
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

	mcpServer, err := newRuntimeServer(ctx, cfg.GRPCAddr)
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
	ticks, stopTicker := newHealthMonitorTicker(defaultHealthMonitorInterval)
	defer stopTicker()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			conn := s.grpcConn()
			if conn == nil {
				log.Printf("gRPC connection is nil, health check skipped")
				continue
			}

			callCtx, cancel := context.WithTimeout(ctx, defaultHealthCheckTimeout)
			status, err := checkConnectionHealth(callCtx, conn)
			cancel()

			if err != nil {
				log.Printf("gRPC health check failed: %v", err)
			} else if status != grpc_health_v1.HealthCheckResponse_SERVING {
				log.Printf("gRPC health check status: %s", status.String())
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

// ServeWithTransport starts the MCP server with an explicit transport.
// This keeps transport lifecycle explicit for integration harnesses that need
// in-memory or custom transports without spawning a separate process.
func (s *Server) ServeWithTransport(ctx context.Context, transport mcp.Transport) error {
	if transport == nil {
		return fmt.Errorf("transport is required")
	}
	return s.serveWithTransport(ctx, transport)
}

// Close releases the gRPC connection held by the server.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	conn := s.takeConn()
	if conn == nil {
		return nil
	}
	if err := closeGRPCConn(conn); err != nil {
		s.restoreConn(conn)
		return err
	}
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
	mcpServer, err := newRuntimeServer(ctx, grpcAddr)
	if err != nil {
		return err
	}
	defer mcpServer.Close()
	return mcpServer.serveWithTransport(ctx, transport)
}

func newRuntimeServer(ctx context.Context, grpcAddr string) (*Server, error) {
	addr := grpcAddress(grpcAddr)
	conn, err := dialGameRuntimeConn(ctx, addr)
	if err != nil {
		return nil, err
	}
	return buildServerFromConn(conn)
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
	conn := s.grpcConn()
	if conn == nil {
		return fmt.Errorf("gRPC connection is not configured")
	}

	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}
	return platformgrpc.WaitForHealth(ctx, conn, "", logf)
}

func (s *Server) grpcConn() *grpc.ClientConn {
	if s == nil {
		return nil
	}
	s.connMu.RLock()
	defer s.connMu.RUnlock()
	return s.conn
}

func (s *Server) takeConn() *grpc.ClientConn {
	if s == nil {
		return nil
	}
	s.connMu.Lock()
	defer s.connMu.Unlock()
	conn := s.conn
	s.conn = nil
	return conn
}

func (s *Server) restoreConn(conn *grpc.ClientConn) {
	if s == nil || conn == nil {
		return
	}
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.conn == nil {
		s.conn = conn
	}
}
