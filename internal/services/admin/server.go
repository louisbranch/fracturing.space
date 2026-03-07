package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	admingrpcdial "github.com/louisbranch/fracturing.space/internal/services/admin/integration/grpcdial"
	"google.golang.org/grpc"
)

// adminAuthzOverrideReason records why admin service uses platform override.
const adminAuthzOverrideReason = "admin_dashboard"

// Config defines the inputs for the admin operator process.
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	AuthAddr        string
	StatusAddr      string
	GRPCDialTimeout time.Duration
	// AuthConfig enables token-based authentication when set.
	AuthConfig *AuthConfig
}

// Server hosts the admin dashboard and manages authenticated gRPC clients.
type Server struct {
	httpAddr    string
	grpcAddr    string
	authAddr    string
	grpcClients *grpcClients
	statusConn  *grpc.ClientConn
	httpServer  *http.Server
}

// grpcClients stores gRPC connections and clients for the admin server.
type grpcClients struct {
	mu                sync.RWMutex
	gameConn          *grpc.ClientConn
	authConn          *grpc.ClientConn
	authClient        authv1.AuthServiceClient
	accountClient     authv1.AccountServiceClient
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	contentClient     daggerheartv1.DaggerheartContentServiceClient
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	characterClient   statev1.CharacterServiceClient
	participantClient statev1.ParticipantServiceClient
	inviteClient      statev1.InviteServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	eventClient       statev1.EventServiceClient
	statisticsClient  statev1.StatisticsServiceClient
	systemClient      statev1.SystemServiceClient
}

// gameGRPCClients holds all game clients created by a successful game dial.
type gameGRPCClients struct {
	conn              *grpc.ClientConn
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	contentClient     daggerheartv1.DaggerheartContentServiceClient
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	characterClient   statev1.CharacterServiceClient
	participantClient statev1.ParticipantServiceClient
	inviteClient      statev1.InviteServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	eventClient       statev1.EventServiceClient
	statisticsClient  statev1.StatisticsServiceClient
	systemClient      statev1.SystemServiceClient
}

// authGRPCClients holds all auth service clients created by auth dial.
type authGRPCClients struct {
	conn          *grpc.ClientConn
	authClient    authv1.AuthServiceClient
	accountClient authv1.AccountServiceClient
}

// CampaignClient returns the current campaign client.
func (g *grpcClients) CampaignClient() statev1.CampaignServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.campaignClient
}

// AuthClient returns the current auth client.
func (g *grpcClients) AuthClient() authv1.AuthServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.authClient
}

// AccountClient returns the current account client.
func (g *grpcClients) AccountClient() authv1.AccountServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.accountClient
}

// SessionClient returns the current session client.
func (g *grpcClients) SessionClient() statev1.SessionServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sessionClient
}

// CharacterClient returns the current character client.
func (g *grpcClients) CharacterClient() statev1.CharacterServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.characterClient
}

// ParticipantClient returns the current participant client.
func (g *grpcClients) ParticipantClient() statev1.ParticipantServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.participantClient
}

// InviteClient returns the current invite client.
func (g *grpcClients) InviteClient() statev1.InviteServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.inviteClient
}

// SnapshotClient returns the current snapshot client.
func (g *grpcClients) SnapshotClient() statev1.SnapshotServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.snapshotClient
}

// EventClient returns the current event client.
func (g *grpcClients) EventClient() statev1.EventServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.eventClient
}

// StatisticsClient returns the current statistics client.
func (g *grpcClients) StatisticsClient() statev1.StatisticsServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.statisticsClient
}

// SystemClient returns the current system client.
func (g *grpcClients) SystemClient() statev1.SystemServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.systemClient
}

// DaggerheartContentClient returns the Daggerheart content client.
func (g *grpcClients) DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.contentClient
}

// HasGameConnection reports whether a game gRPC connection is already set.
func (g *grpcClients) HasGameConnection() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.gameConn != nil
}

// HasAuthConnection reports whether an auth gRPC connection is already set.
func (g *grpcClients) HasAuthConnection() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.authConn != nil
}

// SetGameClients stores the game gRPC connection and clients after first successful dial.
func (g *grpcClients) SetGameClients(clients gameGRPCClients) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gameConn != nil {
		return
	}
	g.gameConn = clients.conn
	g.daggerheartClient = clients.daggerheartClient
	g.contentClient = clients.contentClient
	g.campaignClient = clients.campaignClient
	g.sessionClient = clients.sessionClient
	g.characterClient = clients.characterClient
	g.participantClient = clients.participantClient
	g.inviteClient = clients.inviteClient
	g.snapshotClient = clients.snapshotClient
	g.eventClient = clients.eventClient
	g.statisticsClient = clients.statisticsClient
	g.systemClient = clients.systemClient
}

// SetAuthClients stores the auth gRPC connection and client.
func (g *grpcClients) SetAuthClients(clients authGRPCClients) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.authConn != nil {
		return
	}
	g.authConn = clients.conn
	g.authClient = clients.authClient
	g.accountClient = clients.accountClient
}

// Close releases any gRPC resources held by the clients.
func (g *grpcClients) Close() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gameConn != nil {
		if err := g.gameConn.Close(); err != nil {
			log.Printf("close admin game gRPC connection: %v", err)
		}
		g.gameConn = nil
	}
	if g.authConn != nil {
		if err := g.authConn.Close(); err != nil {
			log.Printf("close admin auth gRPC connection: %v", err)
		}
		g.authConn = nil
	}
}

// NewServer builds a configured admin server.
func NewServer(ctx context.Context, cfg Config) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}

	clients := &grpcClients{}
	if strings.TrimSpace(cfg.GRPCAddr) != "" {
		clientsResult, err := dialGameGRPC(ctx, cfg)
		if err != nil {
			log.Printf("admin game gRPC dial failed: %v", err)
			go connectGameGRPCWithRetry(ctx, cfg, clients)
		} else {
			clients.SetGameClients(clientsResult)
		}
	}
	if strings.TrimSpace(cfg.AuthAddr) != "" {
		clientsResult, err := dialAuthGRPC(ctx, cfg)
		if err != nil {
			log.Printf("admin auth gRPC dial failed: %v", err)
			go connectAuthGRPCWithRetry(ctx, cfg, clients)
		} else {
			clients.SetAuthClients(clientsResult)
		}
	}

	// Status service client for the status admin module.
	var statusClient statusv1.StatusServiceClient
	var statusConn *grpc.ClientConn
	if strings.TrimSpace(cfg.StatusAddr) != "" {
		statusConn = platformgrpc.DialLenient(ctx, cfg.StatusAddr, log.Printf)
		if statusConn != nil {
			statusClient = statusv1.NewStatusServiceClient(statusConn)
		}
	}

	handler := NewHandlerWithConfig(clients, cfg.GRPCAddr, cfg.AuthConfig, statusClient)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return &Server{
		httpAddr:    httpAddr,
		grpcAddr:    cfg.GRPCAddr,
		authAddr:    cfg.AuthAddr,
		grpcClients: clients,
		statusConn:  statusConn,
		httpServer:  httpServer,
	}, nil
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("admin server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	serveErr := make(chan error, 1)
	log.Printf("admin listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}

// Close releases any gRPC resources held by the server.
func (s *Server) Close() {
	if s == nil {
		return
	}
	if s.grpcClients != nil {
		s.grpcClients.Close()
	}
	if s.statusConn != nil {
		if err := s.statusConn.Close(); err != nil {
			log.Printf("close admin status gRPC connection: %v", err)
		}
	}
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
}

// dialGameGRPC connects to the game server and returns a client.
func dialGameGRPC(ctx context.Context, cfg Config) (gameGRPCClients, error) {
	clients, err := admingrpcdial.DialGame(ctx, cfg.GRPCAddr, cfg.GRPCDialTimeout, adminAuthzOverrideReason)
	if err != nil {
		return gameGRPCClients{}, err
	}
	return gameGRPCClients{
		conn:              clients.Conn,
		daggerheartClient: clients.DaggerheartClient,
		contentClient:     clients.ContentClient,
		campaignClient:    clients.CampaignClient,
		sessionClient:     clients.SessionClient,
		characterClient:   clients.CharacterClient,
		participantClient: clients.ParticipantClient,
		inviteClient:      clients.InviteClient,
		snapshotClient:    clients.SnapshotClient,
		eventClient:       clients.EventClient,
		statisticsClient:  clients.StatisticsClient,
		systemClient:      clients.SystemClient,
	}, nil
}

// dialAuthGRPC connects to the auth server and returns a client.
func dialAuthGRPC(ctx context.Context, cfg Config) (authGRPCClients, error) {
	clients, err := admingrpcdial.DialAuth(ctx, cfg.AuthAddr, cfg.GRPCDialTimeout)
	if err != nil {
		return authGRPCClients{}, err
	}
	return authGRPCClients{
		conn:          clients.Conn,
		authClient:    clients.AuthClient,
		accountClient: clients.AccountClient,
	}, nil
}

// connectGameGRPCWithRetry keeps dialing until a connection is established or context ends.
func connectGameGRPCWithRetry(ctx context.Context, cfg Config, clients *grpcClients) {
	if clients == nil {
		return
	}
	if strings.TrimSpace(cfg.GRPCAddr) == "" {
		return
	}
	connectGRPCWithRetry(
		ctx,
		cfg.GRPCAddr,
		clients.HasGameConnection,
		func(connectCtx context.Context) error {
			clientsResult, err := dialGameGRPC(connectCtx, cfg)
			if err != nil {
				return err
			}
			clients.SetGameClients(clientsResult)
			return nil
		},
		"admin gRPC connected to %s",
		"admin game gRPC dial failed: %v",
	)
}

// connectAuthGRPCWithRetry keeps dialing until a connection is established or context ends.
func connectAuthGRPCWithRetry(ctx context.Context, cfg Config, clients *grpcClients) {
	if clients == nil {
		return
	}
	if strings.TrimSpace(cfg.AuthAddr) == "" {
		return
	}
	connectGRPCWithRetry(
		ctx,
		cfg.AuthAddr,
		clients.HasAuthConnection,
		func(connectCtx context.Context) error {
			authClients, err := dialAuthGRPC(connectCtx, cfg)
			if err != nil {
				return err
			}
			clients.SetAuthClients(authClients)
			return nil
		},
		"admin auth gRPC connected to %s",
		"admin auth gRPC dial failed: %v",
	)
}

func connectGRPCWithRetry(
	ctx context.Context,
	address string,
	hasConnection func() bool,
	connect func(context.Context) error,
	successLogFormat string,
	failureLogFormat string,
) {
	admingrpcdial.ConnectWithRetry(ctx, address, hasConnection, connect, successLogFormat, failureLogFormat)
}
