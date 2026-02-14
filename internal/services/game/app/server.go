package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// serverEnv holds env-parsed configuration for the game server.
type serverEnv struct {
	AuthAddr          string `env:"FRACTURING_SPACE_AUTH_ADDR"                 envDefault:"localhost:8083"`
	EventsDBPath      string `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath string `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	ContentDBPath     string `env:"FRACTURING_SPACE_GAME_CONTENT_DB_PATH"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = env.Parse(&cfg)
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	if cfg.ContentDBPath == "" {
		cfg.ContentDBPath = filepath.Join("data", "game-content.db")
	}
	return cfg
}

// Server hosts the game service.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	stores     *storageBundle
	authConn   *grpc.ClientConn
}

// storageBundle groups the three SQLite stores and manages their lifecycle.
type storageBundle struct {
	events      *storagesqlite.Store
	projections *storagesqlite.Store
	content     *storagesqlite.Store
}

// Close closes all stores in the bundle, logging any errors.
func (b *storageBundle) Close() {
	if b == nil {
		return
	}
	if b.events != nil {
		if err := b.events.Close(); err != nil {
			log.Printf("close event store: %v", err)
		}
	}
	if b.projections != nil {
		if err := b.projections.Close(); err != nil {
			log.Printf("close projection store: %v", err)
		}
	}
	if b.content != nil {
		if err := b.content.Close(); err != nil {
			log.Printf("close content store: %v", err)
		}
	}
}

// New creates a configured game server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured game server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	srvEnv := loadServerEnv()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	bundle, err := openStorageBundle(srvEnv)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	stores := gamegrpc.Stores{
		Campaign:           bundle.projections,
		Participant:        bundle.projections,
		ClaimIndex:         bundle.projections,
		Invite:             bundle.projections,
		Character:          bundle.projections,
		Daggerheart:        bundle.projections,
		Session:            bundle.projections,
		SessionGate:        bundle.projections,
		SessionSpotlight:   bundle.projections,
		Event:              bundle.events,
		Telemetry:          bundle.events,
		Statistics:         bundle.projections,
		Outcome:            bundle.events,
		Snapshot:           bundle.projections,
		CampaignFork:       bundle.projections,
		DaggerheartContent: bundle.content,
	}
	if err := stores.Validate(); err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("validate stores: %w", err)
	}

	authConn, authClient, err := dialAuthGRPC(context.Background(), srvEnv.AuthAddr)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmeta.UnaryServerInterceptor(nil),
			interceptors.TelemetryInterceptor(bundle.events),
			interceptors.SessionLockInterceptor(bundle.projections),
		),
		grpc.StreamInterceptor(grpcmeta.StreamServerInterceptor(nil)),
	)
	daggerheartStores := daggerheartservice.Stores{
		Campaign:           bundle.projections,
		Character:          bundle.projections,
		Session:            bundle.projections,
		SessionGate:        bundle.projections,
		SessionSpotlight:   bundle.projections,
		Daggerheart:        bundle.projections,
		DaggerheartContent: bundle.content,
		Event:              bundle.events,
	}
	daggerheartService := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	contentService := daggerheartservice.NewDaggerheartContentService(daggerheartStores)
	campaignService := gamegrpc.NewCampaignServiceWithAuth(stores, authClient)
	participantService := gamegrpc.NewParticipantService(stores)
	inviteService := gamegrpc.NewInviteServiceWithAuth(stores, authClient)
	characterService := gamegrpc.NewCharacterService(stores)
	snapshotService := gamegrpc.NewSnapshotService(stores)
	sessionService := gamegrpc.NewSessionService(stores)
	forkService := gamegrpc.NewForkService(stores)
	eventService := gamegrpc.NewEventService(stores)
	statisticsService := gamegrpc.NewStatisticsService(stores)
	systemService := gamegrpc.NewSystemService(nil)
	healthServer := health.NewServer()
	daggerheartv1.RegisterDaggerheartServiceServer(grpcServer, daggerheartService)
	daggerheartv1.RegisterDaggerheartContentServiceServer(grpcServer, contentService)
	statev1.RegisterCampaignServiceServer(grpcServer, campaignService)
	statev1.RegisterParticipantServiceServer(grpcServer, participantService)
	statev1.RegisterInviteServiceServer(grpcServer, inviteService)
	statev1.RegisterCharacterServiceServer(grpcServer, characterService)
	statev1.RegisterSnapshotServiceServer(grpcServer, snapshotService)
	statev1.RegisterSessionServiceServer(grpcServer, sessionService)
	statev1.RegisterForkServiceServer(grpcServer, forkService)
	statev1.RegisterEventServiceServer(grpcServer, eventService)
	statev1.RegisterStatisticsServiceServer(grpcServer, statisticsService)
	statev1.RegisterSystemServiceServer(grpcServer, systemService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartContentService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ParticipantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.InviteService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CharacterService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SnapshotService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ForkService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.EventService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.StatisticsService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SystemService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		stores:     bundle,
		authConn:   authConn,
	}, nil
}

// Addr returns the listener address for the game server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a game server until the context ends.
func Run(ctx context.Context, port int) error {
	grpcServer, err := New(port)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// RunWithAddr creates and serves a game server until the context ends.
func RunWithAddr(ctx context.Context, addr string) error {
	grpcServer, err := NewWithAddr(addr)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts the game server and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.closeResources()

	log.Printf("game server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	handleErr := func(err error) error {
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}

	select {
	case <-ctx.Done():
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
		err := <-serveErr
		return handleErr(err)
	case err := <-serveErr:
		return handleErr(err)
	}
}

// openStorageBundle opens the events, projections, and content stores as a unit.
func openStorageBundle(srvEnv serverEnv) (*storageBundle, error) {
	eventStore, err := openEventStore(srvEnv.EventsDBPath)
	if err != nil {
		return nil, err
	}
	projStore, err := openProjectionStore(srvEnv.ProjectionsDBPath)
	if err != nil {
		_ = eventStore.Close()
		return nil, err
	}
	contentStore, err := openContentStore(srvEnv.ContentDBPath)
	if err != nil {
		_ = eventStore.Close()
		_ = projStore.Close()
		return nil, err
	}
	return &storageBundle{
		events:      eventStore,
		projections: projStore,
		content:     contentStore,
	}, nil
}

func dialAuthGRPC(ctx context.Context, authAddr string) (*grpc.ClientConn, authv1.AuthServiceClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeouts.GRPCDial)
	defer cancel()
	conn, err := grpc.DialContext(
		dialCtx,
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial auth gRPC %s: %w", authAddr, err)
	}
	logf := func(format string, args ...any) {
		log.Printf("auth %s", fmt.Sprintf(format, args...))
	}
	if err := platformgrpc.WaitForHealth(ctx, conn, "", logf); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("auth gRPC health check failed for %s: %w", authAddr, err)
	}
	return conn, authv1.NewAuthServiceClient(conn), nil
}

func openEventStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenEvents(path, keyring)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("verify event integrity: %w", err)
	}
	return store, nil
}

func openProjectionStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenProjections(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	return store, nil
}

func openContentStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenContent(path)
	if err != nil {
		return nil, fmt.Errorf("open content store: %w", err)
	}
	return store, nil
}

func ensureDir(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create storage dir: %w", err)
		}
	}
	return nil
}

// closeResources releases all server resources.
func (s *Server) closeResources() {
	if s == nil {
		return
	}
	s.stores.Close()
	if s.authConn != nil {
		if err := s.authConn.Close(); err != nil {
			log.Printf("close auth conn: %v", err)
		}
	}
}
