package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
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

// Server hosts the game service.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	eventStore *storagesqlite.Store
	projStore  *storagesqlite.Store
	authConn   *grpc.ClientConn
}

// defaultAuthDialTimeout caps auth gRPC dial wait time.
const defaultAuthDialTimeout = 2 * time.Second

// defaultAuthAddr defines the fallback auth gRPC address.
const defaultAuthAddr = "localhost:8083"

// New creates a configured game server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured game server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	eventStore, projStore, err := openStores()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	stores := gamegrpc.Stores{
		Campaign:     projStore,
		Participant:  projStore,
		ClaimIndex:   projStore,
		Invite:       projStore,
		Character:    projStore,
		Daggerheart:  projStore,
		Session:      projStore,
		Event:        eventStore,
		Telemetry:    eventStore,
		Statistics:   projStore,
		Outcome:      eventStore,
		Snapshot:     projStore,
		CampaignFork: projStore,
	}

	authConn, authClient, err := dialAuthGRPC(context.Background())
	if err != nil {
		_ = listener.Close()
		_ = eventStore.Close()
		_ = projStore.Close()
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmeta.UnaryServerInterceptor(nil),
			interceptors.TelemetryInterceptor(eventStore),
			interceptors.SessionLockInterceptor(projStore),
		),
		grpc.StreamInterceptor(grpcmeta.StreamServerInterceptor(nil)),
	)
	daggerheartStores := daggerheartservice.Stores{
		Campaign:    projStore,
		Character:   projStore,
		Session:     projStore,
		Daggerheart: projStore,
		Event:       eventStore,
	}
	daggerheartService := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	campaignService := gamegrpc.NewCampaignServiceWithAuth(stores, authClient)
	participantService := gamegrpc.NewParticipantService(stores)
	inviteService := gamegrpc.NewInviteService(stores)
	characterService := gamegrpc.NewCharacterService(stores)
	snapshotService := gamegrpc.NewSnapshotService(stores)
	sessionService := gamegrpc.NewSessionService(stores)
	forkService := gamegrpc.NewForkService(stores)
	eventService := gamegrpc.NewEventService(stores)
	statisticsService := gamegrpc.NewStatisticsService(stores)
	healthServer := health.NewServer()
	daggerheartv1.RegisterDaggerheartServiceServer(grpcServer, daggerheartService)
	statev1.RegisterCampaignServiceServer(grpcServer, campaignService)
	statev1.RegisterParticipantServiceServer(grpcServer, participantService)
	statev1.RegisterInviteServiceServer(grpcServer, inviteService)
	statev1.RegisterCharacterServiceServer(grpcServer, characterService)
	statev1.RegisterSnapshotServiceServer(grpcServer, snapshotService)
	statev1.RegisterSessionServiceServer(grpcServer, sessionService)
	statev1.RegisterForkServiceServer(grpcServer, forkService)
	statev1.RegisterEventServiceServer(grpcServer, eventService)
	statev1.RegisterStatisticsServiceServer(grpcServer, statisticsService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ParticipantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.InviteService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CharacterService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SnapshotService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ForkService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.EventService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.StatisticsService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		eventStore: eventStore,
		projStore:  projStore,
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
	defer s.closeStores()

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

func openStores() (*storagesqlite.Store, *storagesqlite.Store, error) {
	eventStore, err := openEventStore()
	if err != nil {
		return nil, nil, err
	}
	projStore, err := openProjectionStore()
	if err != nil {
		_ = eventStore.Close()
		return nil, nil, err
	}
	return eventStore, projStore, nil
}

func dialAuthGRPC(ctx context.Context) (*grpc.ClientConn, authv1.AuthServiceClient, error) {
	authAddr := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AUTH_ADDR"))
	if authAddr == "" {
		authAddr = defaultAuthAddr
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dialCtx, cancel := context.WithTimeout(ctx, defaultAuthDialTimeout)
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

func openEventStore() (*storagesqlite.Store, error) {
	path := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH"))
	if path == "" {
		path = filepath.Join("data", "game-events.db")
	}
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

func openProjectionStore() (*storagesqlite.Store, error) {
	path := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"))
	if path == "" {
		path = filepath.Join("data", "game-projections.db")
	}
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenProjections(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
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

func (s *Server) closeStores() {
	if s == nil {
		return
	}
	if s.eventStore != nil {
		if err := s.eventStore.Close(); err != nil {
			log.Printf("close event store: %v", err)
		}
	}
	if s.projStore != nil {
		if err := s.projStore.Close(); err != nil {
			log.Printf("close projection store: %v", err)
		}
	}
	if s.authConn != nil {
		if err := s.authConn.Close(); err != nil {
			log.Printf("close auth conn: %v", err)
		}
	}
}
