package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"sync"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/conformance"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
)

const (
	// serverVersion identifies the MCP server version.
	serverVersion = "0.1.0"
)

// registration-related module composition moved to server_registration.go.

// mcpServerRegistrationAdapter and registration client wiring moved to server_registration.go.

// mcpToolRegistrar and module registration assembly moved to server_registration.go.

// serverName identifies this MCP server to clients.
var serverName = branding.AppName + " MCP"

var (
	newManagedConn         = platformgrpc.NewManagedConn
	buildMCPServerFromConn = newServer
)

// RegistrationProfile controls which MCP registrations are exposed by a
// server instance. The default runtime uses the standard profile; integration
// harnesses opt into broader bootstrap tools explicitly.
type RegistrationProfile string

const (
	RegistrationProfileStandard RegistrationProfile = "standard"
	RegistrationProfileHarness  RegistrationProfile = "harness"
)

// Config configures the MCP server.
type Config struct {
	GRPCAddr            string
	AIAddr              string
	HTTPAddr            string // HTTP server address (e.g., "localhost:8085"). Defaults to localhost:8085.
	TLSConfig           *tls.Config
	RegistrationProfile RegistrationProfile
}

// Server hosts the MCP server.
type Server struct {
	mcpServer *mcp.Server
	gameMc    *platformgrpc.ManagedConn
	aiMc      *platformgrpc.ManagedConn
	profile   mcpRegistrationProfile
	ctx       sessionctx.Context
	ctxMu     sync.RWMutex
}

// New creates a configured MCP server that connects to state and game system
// gRPC services and hydrates tool/resource handlers from those APIs.
func New(grpcAddr string) (*Server, error) {
	return buildServerWithManagedConns(context.Background(), grpcAddr, "", platformgrpc.ModeOptional, mcpRegistrationProfileStandard)
}

// NewHarness creates a non-production MCP server with mutable context
// bootstrap tooling enabled for integration harnesses and focused local tests.
func NewHarness(grpcAddr string) (*Server, error) {
	return buildServerWithManagedConns(context.Background(), grpcAddr, "", platformgrpc.ModeOptional, mcpRegistrationProfileHarness)
}

// newServer creates MCP tool/resource handler bindings once and keeps shared
// context for protocol state updates.
func newServer(conn *grpc.ClientConn) (*Server, error) {
	return newServerWithAIConnProfile(conn, nil, mcpRegistrationProfileStandard, sessionctx.Context{})
}

func newServerWithAIConn(conn *grpc.ClientConn, aiConn *grpc.ClientConn) (*Server, error) {
	return newServerWithAIConnProfile(conn, aiConn, mcpRegistrationProfileStandard, sessionctx.Context{})
}

func newInternalAISessionServer(conn *grpc.ClientConn, aiConn *grpc.ClientConn, sessionCtx mcpbridge.SessionContext) (*Server, error) {
	return newServerWithAIConnProfile(conn, aiConn, mcpRegistrationProfileInternalAI, sessionctx.Context{
		CampaignID:    sessionCtx.CampaignID,
		SessionID:     sessionCtx.SessionID,
		ParticipantID: sessionCtx.ParticipantID,
	})
}

func newServerWithAIConnProfile(conn *grpc.ClientConn, aiConn *grpc.ClientConn, profile mcpRegistrationProfile, initialContext sessionctx.Context) (*Server, error) {
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
	sceneClient := statev1.NewSceneServiceClient(conn)
	interactionClient := statev1.NewInteractionServiceClient(conn)
	forkClient := statev1.NewForkServiceClient(conn)
	eventClient := statev1.NewEventServiceClient(conn)
	var campaignArtifactClient aiv1.CampaignArtifactServiceClient
	var systemReferenceClient aiv1.SystemReferenceServiceClient
	if aiConn != nil {
		campaignArtifactClient = aiv1.NewCampaignArtifactServiceClient(aiConn)
		systemReferenceClient = aiv1.NewSystemReferenceServiceClient(aiConn)
	}

	server := &Server{mcpServer: mcpServer, profile: profile, ctx: initialContext}
	resourceNotifier := func(ctx context.Context, uri string) {
		if strings.TrimSpace(uri) == "" {
			return
		}
		if ctx == nil || ctx.Err() != nil {
			ctx = context.Background()
		}
		if err := mcpServer.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}); err != nil {
			log.Printf("mcp resource updated notify failed: uri=%s err=%v", uri, err)
		}
	}

	for _, module := range newMCPRegistrationModules(
		server,
		mcpRegistrationClients{
			daggerheartClient:      daggerheartClient,
			campaignClient:         campaignClient,
			participantClient:      participantClient,
			characterClient:        characterClient,
			snapshotClient:         snapshotClient,
			sessionClient:          sessionClient,
			sceneClient:            sceneClient,
			interactionClient:      interactionClient,
			forkClient:             forkClient,
			eventClient:            eventClient,
			campaignArtifactClient: campaignArtifactClient,
			systemReferenceClient:  systemReferenceClient,
		},
		profile,
		resourceNotifier,
	) {
		if err := module.register(mcpServerRegistrationAdapter{server: mcpServer}); err != nil {
			return nil, fmt.Errorf("register MCP module %q: %w", module.name, err)
		}
	}

	conformance.Register(mcpServer)

	return server, nil
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close mcp %s managed conn: %v", name, err)
	}
}

// runtime and transport handlers moved to server_runtime.go.

// Run and HTTP transport entrypoint moved to server_runtime.go.

// lifecycle, transport, and connection helpers moved to server_runtime.go.
