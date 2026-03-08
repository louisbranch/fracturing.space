package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
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

	// Create HTTP transport with reference to MCP server
	httpTransport := NewHTTPTransportWithServer(httpAddr, mcpServer.mcpServer)
	httpTransport.applyConfig(cfg)

	// Start HTTP server (this will handle all HTTP requests)
	return httpTransport.Start(ctx)
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

// Close releases the managed connection held by the server.
func (s *Server) Close() error {
	if s == nil || s.gameMc == nil {
		return nil
	}
	closeManagedConn(s.gameMc, "game")
	s.gameMc = nil
	return nil
}

// serveWithTransport starts the MCP server using the provided transport.
// The server and its managed connection share a single exit path so cleanup behavior
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
			return fmt.Errorf("close game managed conn: %w", closeErr)
		}
		return fmt.Errorf("serve MCP: %v; close game managed conn: %w", err, closeErr)
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
	return buildServerWithManagedConn(ctx, addr, platformgrpc.ModeRequired)
}

// buildServerWithManagedConn dials the game gRPC service as a ManagedConn and
// wires the MCP server from the resulting connection.
func buildServerWithManagedConn(ctx context.Context, addr string, mode platformgrpc.ManagedConnMode) (*Server, error) {
	mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: addr,
		Mode: mode,
		Logf: log.Printf,
		DialOpts: append(
			platformgrpc.LenientDialOptions(),
			grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(mcpAuthzOverrideReason)),
			grpc.WithChainStreamInterceptor(grpcauthctx.AdminOverrideStreamClientInterceptor(mcpAuthzOverrideReason)),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to game server at %s: %w", addr, err)
	}
	server, err := buildMCPServerFromConn(mc.Conn())
	if err != nil {
		closeManagedConn(mc, "game")
		return nil, err
	}
	server.gameMc = mc
	return server, nil
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
