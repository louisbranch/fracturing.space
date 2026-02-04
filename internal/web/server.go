package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	dualityv1 "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// defaultGRPCDialTimeout caps the dial wait time for gRPC connections.
const defaultGRPCDialTimeout = 2 * time.Second

// Config defines the inputs for the web server.
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	GRPCDialTimeout time.Duration
}

// Server hosts the web client HTTP server and optional gRPC connection.
type Server struct {
	httpAddr   string
	grpcAddr   string
	grpcConn   *grpc.ClientConn
	grpcClient dualityv1.DualityServiceClient
	httpServer *http.Server
}

// NewServer builds a configured web server.
func NewServer(ctx context.Context, config Config) (*Server, error) {
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = defaultGRPCDialTimeout
	}

	handler := NewHandler()
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcConn, grpcClient, err := dialGRPC(ctx, config)
	if err != nil {
		log.Printf("web gRPC dial failed: %v", err)
	}

	return &Server{
		httpAddr:   httpAddr,
		grpcAddr:   config.GRPCAddr,
		grpcConn:   grpcConn,
		grpcClient: grpcClient,
		httpServer: httpServer,
	}, nil
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	serveErr := make(chan error, 1)
	log.Printf("web listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	if s == nil || s.grpcConn == nil {
		return
	}
	if err := s.grpcConn.Close(); err != nil {
		log.Printf("close web gRPC connection: %v", err)
	}
}

// dialGRPC connects to the gRPC server and returns a client.
func dialGRPC(ctx context.Context, config Config) (*grpc.ClientConn, dualityv1.DualityServiceClient, error) {
	grpcAddr := strings.TrimSpace(config.GRPCAddr)
	if grpcAddr == "" {
		return nil, nil, nil
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
		return nil, nil, err
	}

	client := dualityv1.NewDualityServiceClient(conn)
	return conn, client, nil
}
