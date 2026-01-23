package server

import (
	"fmt"
	"log"
	"net"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/louisbranch/duality-protocol/internal/random"
	transportgrpc "github.com/louisbranch/duality-protocol/internal/transport/grpc"
	"google.golang.org/grpc"
)

// Server hosts the Duality gRPC server.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

// New creates a configured gRPC server listening on the provided port.
func New(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	service := transportgrpc.NewDualityService(random.NewSeed)
	pb.RegisterDualityServiceServer(grpcServer, service)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
	}, nil
}

// Serve starts the gRPC server and blocks until it stops.
func (s *Server) Serve() error {
	log.Printf("server listening at %v", s.listener.Addr())
	if err := s.grpcServer.Serve(s.listener); err != nil {
		return fmt.Errorf("serve gRPC: %w", err)
	}
	return nil
}
