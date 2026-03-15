package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/httptransport"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
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
// MCP now serves one internal streamable-HTTP transport for AI orchestration.
func Run(ctx context.Context, cfg Config) error {
	cfg.AIAddr = aiAddress(cfg.AIAddr)
	profile, err := resolveRegistrationProfile(cfg.RegistrationProfile)
	if err != nil {
		return err
	}
	return runWithHTTPTransport(ctx, cfg, profile)
}

// runWithHTTPTransport creates a server and serves it over HTTP transport.
// runWithHTTPTransport keeps HTTP session/stateful transport concerns isolated
// from the same MCP domain handlers used by in-memory and focused harnesses.
func runWithHTTPTransport(ctx context.Context, cfg Config, profile mcpRegistrationProfile) error {
	// Default to localhost-only binding for security
	httpAddr := cfg.HTTPAddr
	if httpAddr == "" {
		httpAddr = "localhost:8085"
	}

	mcpServer, err := newRuntimeServer(ctx, cfg.GRPCAddr, cfg.AIAddr, profile)
	if err != nil {
		return err
	}
	defer mcpServer.Close()

	// Create HTTP transport with reference to the MCP runtime so each HTTP
	// session can bind one fixed internal bridge context when required.
	httpTransport := httptransport.NewHTTPTransportWithRuntime(httpAddr, httpTransportRuntimeFactory{runtime: mcpServer})
	httpTransport.SetTLSConfig(cfg.TLSConfig)

	// Start HTTP server (this will handle all HTTP requests)
	return httpTransport.Start(ctx)
}

// Serve starts the MCP server on a stdio transport and blocks until it stops or
// the context ends. It remains available for focused test harnesses.
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
	if s == nil {
		return nil
	}
	closeManagedConn(s.gameMc, "game")
	closeManagedConn(s.aiMc, "ai")
	s.gameMc = nil
	s.aiMc = nil
	return nil
}

// serveWithTransport starts the MCP server using the provided transport.
// The server and its managed connection share a single exit path so cleanup
// behavior is consistent across HTTP and harness transports.
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

// setContext stores the current server context.
func (s *Server) setContext(ctx sessionctx.Context) {
	if s == nil {
		return
	}
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	s.ctx = ctx
}

// getContext returns the current server context.
func (s *Server) getContext() sessionctx.Context {
	if s == nil {
		return sessionctx.Context{}
	}
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	return s.ctx
}

// runWithTransport creates a server and serves it over the provided transport.
func runWithTransport(ctx context.Context, grpcAddr string, transport mcp.Transport) error {
	return runWithTransportWithAI(ctx, grpcAddr, "", transport)
}

func runWithTransportWithAI(ctx context.Context, grpcAddr string, aiAddr string, transport mcp.Transport) error {
	mcpServer, err := newRuntimeServer(ctx, grpcAddr, aiAddr, mcpRegistrationProfileStandard)
	if err != nil {
		return err
	}
	defer mcpServer.Close()
	return mcpServer.serveWithTransport(ctx, transport)
}

func newRuntimeServer(ctx context.Context, grpcAddr string, aiAddr string, profile mcpRegistrationProfile) (*Server, error) {
	addr := grpcAddress(grpcAddr)
	return buildServerWithManagedConns(ctx, addr, aiAddr, platformgrpc.ModeRequired, profile)
}

// buildServerWithManagedConns dials the game service and optionally the AI service.
func buildServerWithManagedConns(
	ctx context.Context,
	addr string,
	aiAddr string,
	mode platformgrpc.ManagedConnMode,
	profile mcpRegistrationProfile,
) (*Server, error) {
	gameMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: addr,
		Mode: mode,
		Logf: log.Printf,
		DialOpts: append(
			platformgrpc.LenientDialOptions(),
			grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceMCP)),
			grpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServiceMCP)),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to game server at %s: %w", addr, err)
	}
	var aiMc *platformgrpc.ManagedConn
	if strings.TrimSpace(aiAddr) != "" {
		aiMc, err = newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "ai",
			Addr: aiAddr,
			Mode: platformgrpc.ModeOptional,
			Logf: log.Printf,
			DialOpts: append(
				platformgrpc.LenientDialOptions(),
				grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceMCP)),
				grpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServiceMCP)),
			),
		})
		if err != nil {
			log.Printf("mcp: ai managed conn unavailable; ai-backed campaign context disabled: %v", err)
			aiMc = nil
		}
	}
	var server *Server
	if aiMc != nil {
		server, err = newServerWithAIConnProfile(gameMc.Conn(), aiMc.Conn(), profile, sessionctx.Context{})
	} else {
		if profile == mcpRegistrationProfileStandard {
			server, err = buildMCPServerFromConn(gameMc.Conn())
		} else {
			server, err = newServerWithAIConnProfile(gameMc.Conn(), nil, profile, sessionctx.Context{})
		}
	}
	if err != nil {
		closeManagedConn(gameMc, "game")
		closeManagedConn(aiMc, "ai")
		return nil, err
	}
	server.gameMc = gameMc
	server.aiMc = aiMc
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

func aiAddress(fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	if value := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_ADDR")); value != "" {
		return value
	}
	return serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceAI)
}

func resolveRegistrationProfile(profile RegistrationProfile) (mcpRegistrationProfile, error) {
	switch normalized := RegistrationProfile(strings.ToLower(strings.TrimSpace(string(profile)))); normalized {
	case "", RegistrationProfileStandard:
		return mcpRegistrationProfileStandard, nil
	case RegistrationProfileHarness:
		return mcpRegistrationProfileHarness, nil
	default:
		return mcpRegistrationProfileStandard, fmt.Errorf("unsupported MCP registration profile %q", profile)
	}
}
