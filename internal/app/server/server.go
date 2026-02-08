package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	authservice "github.com/louisbranch/fracturing.space/internal/api/grpc/auth"
	"github.com/louisbranch/fracturing.space/internal/api/grpc/interceptors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	stateservice "github.com/louisbranch/fracturing.space/internal/api/grpc/state"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/core/random"
	"github.com/louisbranch/fracturing.space/internal/storage"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Server hosts the Fracturing.Space gRPC server.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      storage.Store
}

// New creates a configured gRPC server listening on the provided port.
func New(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}
	store, err := openCampaignStore()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	stores := stateservice.Stores{
		Campaign:       store,
		Participant:    store,
		Character:      store,
		ControlDefault: store,
		Daggerheart:    store,
		Session:        store,
		Event:          store,
		Telemetry:      store,
		Outcome:        store,
		Snapshot:       store,
		CampaignFork:   store,
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmeta.UnaryServerInterceptor(nil),
			interceptors.TelemetryInterceptor(store),
			interceptors.SessionLockInterceptor(store),
		),
		grpc.StreamInterceptor(grpcmeta.StreamServerInterceptor(nil)),
	)
	daggerheartService := daggerheartservice.NewDaggerheartService(random.NewSeed)
	authService := authservice.NewAuthService(store)
	campaignService := stateservice.NewCampaignService(stores)
	participantService := stateservice.NewParticipantService(stores)
	characterService := stateservice.NewCharacterService(stores)
	snapshotService := stateservice.NewSnapshotService(stores)
	sessionService := stateservice.NewSessionService(stores)
	forkService := stateservice.NewForkService(stores)
	eventService := stateservice.NewEventService(stores)
	healthServer := health.NewServer()
	daggerheartv1.RegisterDaggerheartServiceServer(grpcServer, daggerheartService)
	authv1.RegisterAuthServiceServer(grpcServer, authService)
	statev1.RegisterCampaignServiceServer(grpcServer, campaignService)
	statev1.RegisterParticipantServiceServer(grpcServer, participantService)
	statev1.RegisterCharacterServiceServer(grpcServer, characterService)
	statev1.RegisterSnapshotServiceServer(grpcServer, snapshotService)
	statev1.RegisterSessionServiceServer(grpcServer, sessionService)
	statev1.RegisterForkServiceServer(grpcServer, forkService)
	statev1.RegisterEventServiceServer(grpcServer, eventService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AuthService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.ParticipantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.CharacterService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.SnapshotService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.ForkService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("state.v1.EventService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
	}, nil
}

// Addr returns the listener address for the gRPC server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a gRPC server until the context ends.
func Run(ctx context.Context, port int) error {
	grpcServer, err := New(port)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts the gRPC server and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.closeStore()

	log.Printf("server listening at %v", s.listener.Addr())
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

func openCampaignStore() (storage.Store, error) {
	path := os.Getenv("FRACTURING_SPACE_DB_PATH")
	if path == "" {
		path = filepath.Join("data", "fracturing.space.db")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	store, err := storagesqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store: %w", err)
	}
	return store, nil
}

func (s *Server) closeStore() {
	if s == nil || s.store == nil {
		return
	}
	if err := s.store.Close(); err != nil {
		log.Printf("close campaign store: %v", err)
	}
}
