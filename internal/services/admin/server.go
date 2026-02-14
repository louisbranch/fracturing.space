package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	adminsqlite "github.com/louisbranch/fracturing.space/internal/services/admin/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// adminServerEnv holds env-parsed configuration for the admin server.
type adminServerEnv struct {
	DBPath string `env:"FRACTURING_SPACE_ADMIN_DB_PATH"`
}

func loadAdminServerEnv() adminServerEnv {
	var cfg adminServerEnv
	_ = env.Parse(&cfg)
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join("data", "admin.db")
	}
	return cfg
}

// defaultGRPCRetryDelay sets the initial wait time between gRPC dial attempts.
const defaultGRPCRetryDelay = 500 * time.Millisecond

// maxGRPCRetryDelay caps the backoff between gRPC dial attempts.
const maxGRPCRetryDelay = 10 * time.Second

// Config defines the inputs for the admin server.
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	AuthAddr        string
	GRPCDialTimeout time.Duration
}

// Server hosts the admin dashboard HTTP server and optional gRPC connection.
type Server struct {
	httpAddr    string
	grpcAddr    string
	authAddr    string
	grpcClients *grpcClients
	httpServer  *http.Server
	adminStore  *adminsqlite.Store
}

// grpcClients stores gRPC connections and clients for the admin server.
type grpcClients struct {
	mu                sync.RWMutex
	gameConn          *grpc.ClientConn
	authConn          *grpc.ClientConn
	authClient        authv1.AuthServiceClient
	daggerheartClient daggerheartv1.DaggerheartServiceClient
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

// SetGameConn stores the game gRPC connection and clients.
func (g *grpcClients) SetGameConn(conn *grpc.ClientConn, daggerheartClient daggerheartv1.DaggerheartServiceClient, campaignClient statev1.CampaignServiceClient, sessionClient statev1.SessionServiceClient, characterClient statev1.CharacterServiceClient, participantClient statev1.ParticipantServiceClient, inviteClient statev1.InviteServiceClient, snapshotClient statev1.SnapshotServiceClient, eventClient statev1.EventServiceClient, statisticsClient statev1.StatisticsServiceClient, systemClient statev1.SystemServiceClient) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gameConn != nil {
		return
	}
	g.gameConn = conn
	g.daggerheartClient = daggerheartClient
	g.campaignClient = campaignClient
	g.sessionClient = sessionClient
	g.characterClient = characterClient
	g.participantClient = participantClient
	g.inviteClient = inviteClient
	g.snapshotClient = snapshotClient
	g.eventClient = eventClient
	g.statisticsClient = statisticsClient
	g.systemClient = systemClient
}

// SetAuthConn stores the auth gRPC connection and client.
func (g *grpcClients) SetAuthConn(conn *grpc.ClientConn, authClient authv1.AuthServiceClient) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.authConn != nil {
		return
	}
	g.authConn = conn
	g.authClient = authClient
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
func NewServer(ctx context.Context, config Config) (*Server, error) {
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}

	adminEnv := loadAdminServerEnv()
	adminStore, err := openAdminStore(adminEnv.DBPath)
	if err != nil {
		return nil, err
	}

	clients := &grpcClients{}
	if strings.TrimSpace(config.GRPCAddr) != "" {
		conn, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, inviteClient, snapshotClient, eventClient, statisticsClient, systemClient, err := dialGameGRPC(ctx, config)
		if err != nil {
			log.Printf("admin game gRPC dial failed: %v", err)
			go connectGameGRPCWithRetry(ctx, config, clients)
		} else {
			clients.SetGameConn(conn, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, inviteClient, snapshotClient, eventClient, statisticsClient, systemClient)
		}
	}
	if strings.TrimSpace(config.AuthAddr) != "" {
		conn, authClient, err := dialAuthGRPC(ctx, config)
		if err != nil {
			log.Printf("admin auth gRPC dial failed: %v", err)
			go connectAuthGRPCWithRetry(ctx, config, clients)
		} else {
			clients.SetAuthConn(conn, authClient)
		}
	}

	handler := NewHandlerWithConfig(clients, config.GRPCAddr)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return &Server{
		httpAddr:    httpAddr,
		grpcAddr:    config.GRPCAddr,
		authAddr:    config.AuthAddr,
		grpcClients: clients,
		httpServer:  httpServer,
		adminStore:  adminStore,
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
	if s.adminStore != nil {
		if err := s.adminStore.Close(); err != nil {
			log.Printf("close admin store: %v", err)
		}
	}
}

func openAdminStore(path string) (*adminsqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	store, err := adminsqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open admin sqlite store: %w", err)
	}
	return store, nil
}

// dialGRPC connects to the game server and returns a client.
func dialGameGRPC(ctx context.Context, config Config) (*grpc.ClientConn, daggerheartv1.DaggerheartServiceClient, statev1.CampaignServiceClient, statev1.SessionServiceClient, statev1.CharacterServiceClient, statev1.ParticipantServiceClient, statev1.InviteServiceClient, statev1.SnapshotServiceClient, statev1.EventServiceClient, statev1.StatisticsServiceClient, statev1.SystemServiceClient, error) {
	grpcAddr := strings.TrimSpace(config.GRPCAddr)
	if grpcAddr == "" {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	dialCtx, cancel := context.WithTimeout(ctx, config.GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	logf := func(format string, args ...any) {
		log.Printf("admin game %s", fmt.Sprintf(format, args...))
	}
	if err := platformgrpc.WaitForHealth(ctx, conn, "", logf); err != nil {
		_ = conn.Close()
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("admin game gRPC health check failed for %s: %w", grpcAddr, err)
	}

	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)
	campaignClient := statev1.NewCampaignServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	characterClient := statev1.NewCharacterServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)
	inviteClient := statev1.NewInviteServiceClient(conn)
	snapshotClient := statev1.NewSnapshotServiceClient(conn)
	eventClient := statev1.NewEventServiceClient(conn)
	statisticsClient := statev1.NewStatisticsServiceClient(conn)
	systemClient := statev1.NewSystemServiceClient(conn)
	return conn, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, inviteClient, snapshotClient, eventClient, statisticsClient, systemClient, nil
}

// dialAuthGRPC connects to the auth server and returns a client.
func dialAuthGRPC(ctx context.Context, config Config) (*grpc.ClientConn, authv1.AuthServiceClient, error) {
	authAddr := strings.TrimSpace(config.AuthAddr)
	if authAddr == "" {
		return nil, nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	dialCtx, cancel := context.WithTimeout(ctx, config.GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, err
	}
	logf := func(format string, args ...any) {
		log.Printf("admin auth %s", fmt.Sprintf(format, args...))
	}
	if err := platformgrpc.WaitForHealth(ctx, conn, "", logf); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("admin auth gRPC health check failed for %s: %w", authAddr, err)
	}

	authClient := authv1.NewAuthServiceClient(conn)
	return conn, authClient, nil
}

// connectGameGRPCWithRetry keeps dialing until a connection is established or context ends.
func connectGameGRPCWithRetry(ctx context.Context, config Config, clients *grpcClients) {
	if clients == nil {
		return
	}
	if strings.TrimSpace(config.GRPCAddr) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	retryDelay := defaultGRPCRetryDelay
	for {
		if ctx.Err() != nil {
			return
		}
		if clients.HasGameConnection() {
			return
		}
		conn, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, inviteClient, snapshotClient, eventClient, statisticsClient, systemClient, err := dialGameGRPC(ctx, config)
		if err == nil {
			clients.SetGameConn(conn, daggerheartClient, campaignClient, sessionClient, characterClient, participantClient, inviteClient, snapshotClient, eventClient, statisticsClient, systemClient)
			log.Printf("admin gRPC connected to %s", config.GRPCAddr)
			return
		}
		log.Printf("admin game gRPC dial failed: %v", err)
		timer := time.NewTimer(retryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		if retryDelay < maxGRPCRetryDelay {
			retryDelay *= 2
			if retryDelay > maxGRPCRetryDelay {
				retryDelay = maxGRPCRetryDelay
			}
		}
	}
}

// connectAuthGRPCWithRetry keeps dialing until a connection is established or context ends.
func connectAuthGRPCWithRetry(ctx context.Context, config Config, clients *grpcClients) {
	if clients == nil {
		return
	}
	if strings.TrimSpace(config.AuthAddr) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	retryDelay := defaultGRPCRetryDelay
	for {
		if ctx.Err() != nil {
			return
		}
		if clients.HasAuthConnection() {
			return
		}
		conn, authClient, err := dialAuthGRPC(ctx, config)
		if err == nil {
			clients.SetAuthConn(conn, authClient)
			log.Printf("admin auth gRPC connected to %s", config.AuthAddr)
			return
		}
		log.Printf("admin auth gRPC dial failed: %v", err)
		timer := time.NewTimer(retryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		if retryDelay < maxGRPCRetryDelay {
			retryDelay *= 2
			if retryDelay > maxGRPCRetryDelay {
				retryDelay = maxGRPCRetryDelay
			}
		}
	}
}
