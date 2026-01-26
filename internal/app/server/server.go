package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	pb "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	campaignservice "github.com/louisbranch/duality-engine/internal/campaign/service"
	dualityservice "github.com/louisbranch/duality-engine/internal/duality/service"
	sessionservice "github.com/louisbranch/duality-engine/internal/session/service"
	"github.com/louisbranch/duality-engine/internal/random"
	storagebbolt "github.com/louisbranch/duality-engine/internal/storage/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Server hosts the Duality gRPC server.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *storagebbolt.Store
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

	grpcServer := grpc.NewServer()
	dualityService := dualityservice.NewDualityService(random.NewSeed)
	campaignService := campaignservice.NewCampaignService(campaignservice.Stores{
		Campaign:         store,
		Participant:      store,
		Character:        store,
		CharacterProfile: store,
		CharacterState:   store,
		ControlDefault:   store,
	})
	sessionService := sessionservice.NewSessionService(sessionservice.Stores{
		Campaign: store,
		Session:  store,
	})
	healthServer := health.NewServer()
	pb.RegisterDualityServiceServer(grpcServer, dualityService)
	campaignv1.RegisterCampaignServiceServer(grpcServer, campaignService)
	sessionv1.RegisterSessionServiceServer(grpcServer, sessionService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("duality.v1.DualityService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("campaign.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("session.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)

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

func openCampaignStore() (*storagebbolt.Store, error) {
	path := os.Getenv("DUALITY_DB_PATH")
	if path == "" {
		path = filepath.Join("data", "duality.db")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := storagebbolt.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open campaign store: %w", err)
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
