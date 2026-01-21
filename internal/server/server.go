// Package server provides the gRPC server for dice rolls.
package server

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server hosts the gRPC dice roll service.
type Server struct {
	pb.UnimplementedDiceRollServiceServer
	listener net.Listener
	grpc     *grpc.Server
}

// New creates a configured gRPC server listening on the provided port.
func New(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	server := &Server{
		listener: listener,
		grpc:     grpcServer,
	}
	pb.RegisterDiceRollServiceServer(grpcServer, server)

	return server, nil
}

// Serve starts the gRPC server and blocks until it stops.
func (s *Server) Serve() error {
	log.Printf("server listening at %v", s.listener.Addr())
	if err := s.grpc.Serve(s.listener); err != nil {
		return fmt.Errorf("serve gRPC: %w", err)
	}
	return nil
}

// ActionRoll handles action roll requests.
func (s *Server) ActionRoll(ctx context.Context, in *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ActionRoll is not implemented yet")
}
