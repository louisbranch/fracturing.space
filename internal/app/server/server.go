package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	pb "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	campaignservice "github.com/louisbranch/duality-engine/internal/campaign/service"
	dualityservice "github.com/louisbranch/duality-engine/internal/duality/service"
	"github.com/louisbranch/duality-engine/internal/random"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Server hosts the Duality gRPC server.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
}

// New creates a configured gRPC server listening on the provided port.
func New(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	dualityService := dualityservice.NewDualityService(random.NewSeed)
	campaignService := campaignservice.NewCampaignService()
	healthServer := health.NewServer()
	pb.RegisterDualityServiceServer(grpcServer, dualityService)
	campaignpb.RegisterCampaignServiceServer(grpcServer, campaignService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("duality.v1.DualityService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("campaign.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
	}, nil
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
