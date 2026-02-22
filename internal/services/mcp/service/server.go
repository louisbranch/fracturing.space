package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/conformance"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// serverVersion identifies the MCP server version.
	serverVersion = "0.1.0"
	// mcpAuthzOverrideReason records why MCP uses platform override metadata.
	mcpAuthzOverrideReason = "mcp_service"
)

type mcpRegistrationKind int

const (
	mcpRegistrationKindTools mcpRegistrationKind = iota
	mcpRegistrationKindResources
)

type mcpRegistrationModule struct {
	name     string
	kind     mcpRegistrationKind
	register func(mcpRegistrationTarget) error
}

const (
	mcpDaggerheartToolsModuleName = "daggerheart-tools"
	mcpCampaignToolsModuleName    = "campaign-tools"
	mcpSessionToolsModuleName     = "session-tools"
	mcpForkToolsModuleName        = "fork-tools"
	mcpEventToolsModuleName       = "event-tools"
	mcpContextToolsModuleName     = "context-tools"
	mcpCampaignResourceModuleName = "campaign-resources"
	mcpSessionResourceModuleName  = "session-resources"
	mcpEventResourceModuleName    = "event-resources"
	mcpContextResourceModuleName  = "context-resources"
)

type mcpRegistrationClients struct {
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	campaignClient    statev1.CampaignServiceClient
	participantClient statev1.ParticipantServiceClient
	characterClient   statev1.CharacterServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	sessionClient     statev1.SessionServiceClient
	forkClient        statev1.ForkServiceClient
	eventClient       statev1.EventServiceClient
}

type mcpServerRegistrationAdapter struct {
	server *mcp.Server
}

func (r mcpServerRegistrationAdapter) AddTool(tool *mcp.Tool, handler any) error {
	return addMCPTool(r.server, tool, handler)
}

func (r mcpServerRegistrationAdapter) AddResourceTemplate(resourceTemplate *mcp.ResourceTemplate, handler mcp.ResourceHandler) {
	r.server.AddResourceTemplate(resourceTemplate, handler)
}

func (r mcpServerRegistrationAdapter) AddResource(resource *mcp.Resource, handler mcp.ResourceHandler) {
	r.server.AddResource(resource, handler)
}

type mcpToolRegistrar struct {
	matches func(any) bool
	add     func(*mcp.Server, *mcp.Tool, any)
}

func newMCPToolRegistrar[I any, O any]() mcpToolRegistrar {
	return mcpToolRegistrar{
		matches: func(handler any) bool {
			_, ok := handler.(mcp.ToolHandlerFor[I, O])
			return ok
		},
		add: func(server *mcp.Server, tool *mcp.Tool, handler any) {
			mcp.AddTool(server, tool, handler.(mcp.ToolHandlerFor[I, O]))
		},
	}
}

var mcpToolRegistrars = []mcpToolRegistrar{
	newMCPToolRegistrar[domain.ActionRollInput, domain.ActionRollResult](),
	newMCPToolRegistrar[domain.DualityExplainInput, domain.DualityExplainResult](),
	newMCPToolRegistrar[domain.DualityOutcomeInput, domain.DualityOutcomeResult](),
	newMCPToolRegistrar[domain.DualityProbabilityInput, domain.DualityProbabilityResult](),
	newMCPToolRegistrar[domain.RulesVersionInput, domain.RulesVersionResult](),
	newMCPToolRegistrar[domain.RollDiceInput, domain.RollDiceResult](),
	newMCPToolRegistrar[domain.CampaignCreateInput, domain.CampaignCreateResult](),
	newMCPToolRegistrar[domain.CampaignStatusChangeInput, domain.CampaignStatusResult](),
	newMCPToolRegistrar[domain.ParticipantCreateInput, domain.ParticipantCreateResult](),
	newMCPToolRegistrar[domain.ParticipantUpdateInput, domain.ParticipantUpdateResult](),
	newMCPToolRegistrar[domain.ParticipantDeleteInput, domain.ParticipantDeleteResult](),
	newMCPToolRegistrar[domain.CharacterCreateInput, domain.CharacterCreateResult](),
	newMCPToolRegistrar[domain.CharacterUpdateInput, domain.CharacterUpdateResult](),
	newMCPToolRegistrar[domain.CharacterDeleteInput, domain.CharacterDeleteResult](),
	newMCPToolRegistrar[domain.CharacterControlSetInput, domain.CharacterControlSetResult](),
	newMCPToolRegistrar[domain.CharacterSheetGetInput, domain.CharacterSheetGetResult](),
	newMCPToolRegistrar[domain.CharacterProfilePatchInput, domain.CharacterProfilePatchResult](),
	newMCPToolRegistrar[domain.CharacterStatePatchInput, domain.CharacterStatePatchResult](),
	newMCPToolRegistrar[domain.SessionStartInput, domain.SessionStartResult](),
	newMCPToolRegistrar[domain.SessionEndInput, domain.SessionEndResult](),
	newMCPToolRegistrar[domain.EventListInput, domain.EventListResult](),
	newMCPToolRegistrar[domain.CampaignForkInput, domain.CampaignForkResult](),
	newMCPToolRegistrar[domain.CampaignLineageInput, domain.CampaignLineageResult](),
	newMCPToolRegistrar[domain.SetContextInput, domain.SetContextResult](),
}

func addMCPTool(server *mcp.Server, tool *mcp.Tool, handler any) error {
	for _, registrar := range mcpToolRegistrars {
		if registrar.matches(handler) {
			registrar.add(server, tool, handler)
			return nil
		}
	}
	toolName := "<nil>"
	if tool != nil {
		toolName = tool.Name
	}
	return fmt.Errorf("mcp registration adapter does not support handler type %T for tool %q", handler, toolName)
}

func newMCPRegistrationModules(
	server *Server,
	clients mcpRegistrationClients,
	notify domain.ResourceUpdateNotifier,
) []mcpRegistrationModule {
	return []mcpRegistrationModule{
		{
			name: mcpDaggerheartToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerDaggerheartTools(registrar, clients.daggerheartClient)
			},
		},
		{
			name: mcpCampaignToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerCampaignTools(registrar, clients.campaignClient, clients.participantClient, clients.characterClient, clients.snapshotClient, server.getContext, notify)
			},
		},
		{
			name: mcpSessionToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerSessionTools(registrar, clients.sessionClient, server.getContext, notify)
			},
		},
		{
			name: mcpForkToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerForkTools(registrar, clients.forkClient, notify)
			},
		},
		{
			name: mcpEventToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerEventTools(registrar, clients.eventClient, server.getContext)
			},
		},
		{
			name: mcpContextToolsModuleName,
			kind: mcpRegistrationKindTools,
			register: func(registrar mcpRegistrationTarget) error {
				return registerContextTools(registrar, clients.campaignClient, clients.sessionClient, clients.participantClient, server, notify)
			},
		},
		{
			name: mcpCampaignResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerCampaignResources(registrar, clients.campaignClient, clients.participantClient, clients.characterClient)
				return nil
			},
		},
		{
			name: mcpSessionResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerSessionResources(registrar, clients.sessionClient)
				return nil
			},
		},
		{
			name: mcpEventResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerEventResources(registrar, clients.eventClient)
				return nil
			},
		},
		{
			name: mcpContextResourceModuleName,
			kind: mcpRegistrationKindResources,
			register: func(registrar mcpRegistrationTarget) error {
				registerContextResources(registrar, server)
				return nil
			},
		},
	}
}

// serverName identifies this MCP server to clients.
var serverName = branding.AppName + " MCP"

// TransportKind identifies the MCP transport implementation.
type TransportKind string

const (
	// TransportStdio uses standard input/output for MCP.
	TransportStdio TransportKind = "stdio"
	// TransportHTTP runs MCP over HTTP/SSE for browser or remote clients.
	TransportHTTP TransportKind = "http"
)

// Config configures the MCP server.
type Config struct {
	GRPCAddr  string
	Transport TransportKind
	HTTPAddr  string // HTTP server address (e.g., "localhost:8081"). Defaults to localhost:8081 for HTTP transport.
	// TODO: Add TLSConfig field for future TLS/HTTPS support
	// Current deployments are expected to run MCP over trusted local transport only.
	// This means HTTPS and transport-level trust are intentionally out-of-scope for now,
	// and must be added before any non-local production exposure.
	// TODO: Add AuthToken field for future API key authentication
	// Authentication is intentionally deferred in this config because MCP transport
	// defaults to local development trust assumptions; API tokens should be required
	// before opening this server beyond that boundary.
}

// Server hosts the MCP server.
type Server struct {
	mcpServer *mcp.Server
	conn      *grpc.ClientConn
	ctx       domain.Context
	ctxMu     sync.RWMutex
}

// New creates a configured MCP server that connects to state and game system
// gRPC services and hydrates tool/resource handlers from those APIs.
func New(grpcAddr string) (*Server, error) {
	addr := grpcAddress(grpcAddr)
	conn, err := newGRPCConn(addr)
	if err != nil {
		return nil, fmt.Errorf("connect to game server at %s: %w", addr, err)
	}
	return newServer(conn)
}

// newServer creates MCP tool/resource handler bindings once and keeps shared
// context for protocol state updates.
func newServer(conn *grpc.ClientConn) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, &mcp.ServerOptions{
		CompletionHandler:  completionHandler,
		SubscribeHandler:   resourceSubscribeHandler,
		UnsubscribeHandler: resourceUnsubscribeHandler,
	})

	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)
	campaignClient := statev1.NewCampaignServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)
	characterClient := statev1.NewCharacterServiceClient(conn)
	snapshotClient := statev1.NewSnapshotServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	forkClient := statev1.NewForkServiceClient(conn)
	eventClient := statev1.NewEventServiceClient(conn)

	server := &Server{mcpServer: mcpServer, conn: conn}
	resourceNotifier := func(ctx context.Context, uri string) {
		if strings.TrimSpace(uri) == "" {
			return
		}
		if ctx == nil {
			ctx = context.Background()
		}
		if err := mcpServer.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}); err != nil {
			log.Printf("mcp resource updated notify failed: uri=%s err=%v", uri, err)
		}
	}

	for _, module := range newMCPRegistrationModules(
		server,
		mcpRegistrationClients{
			daggerheartClient: daggerheartClient,
			campaignClient:    campaignClient,
			participantClient: participantClient,
			characterClient:   characterClient,
			snapshotClient:    snapshotClient,
			sessionClient:     sessionClient,
			forkClient:        forkClient,
			eventClient:       eventClient,
		},
		resourceNotifier,
	) {
		if err := module.register(mcpServerRegistrationAdapter{server: mcpServer}); err != nil {
			return nil, fmt.Errorf("register MCP module %q: %w", module.name, err)
		}
	}

	conformance.Register(mcpServer)

	return server, nil
}

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
