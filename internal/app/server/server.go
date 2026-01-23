package server

import (
	"context"
	"errors"
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
		s.grpcServer.GracefulStop()
		err := <-serveErr
		return handleErr(err)
	case err := <-serveErr:
		return handleErr(err)
	}
}
