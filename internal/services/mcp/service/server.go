package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/conformance"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
)

const (
	// serverVersion identifies the MCP server version.
	serverVersion = "0.1.0"
	// mcpAuthzOverrideReason records why MCP uses platform override metadata.
	mcpAuthzOverrideReason = "mcp_service"
)

// registration-related module composition moved to server_registration.go.

// mcpServerRegistrationAdapter and registration client wiring moved to server_registration.go.

// mcpToolRegistrar and module registration assembly moved to server_registration.go.

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
	AuthToken string // Optional bearer token accepted by /mcp endpoints when OAuth is also configured.
	TLSConfig *tls.Config

	// Optional extension points for request admission and fairness controls.
	RequestAuthorizer RequestAuthorizer
	RateLimiter       RequestRateLimiter
}

// RequestAuthorizer validates incoming MCP HTTP requests before message handling.
type RequestAuthorizer interface {
	Authorize(r *http.Request) error
}

// RequestRateLimiter throttles incoming MCP HTTP requests.
type RequestRateLimiter interface {
	// Allow returns an error when the request should be rejected.
	Allow(r *http.Request) error
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

// runtime and transport handlers moved to server_runtime.go.

// Run and HTTP transport entrypoint moved to server_runtime.go.

// lifecycle, transport, and connection helpers moved to server_runtime.go.
