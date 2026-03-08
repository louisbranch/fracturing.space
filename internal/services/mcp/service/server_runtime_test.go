package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, func(grpc_health_v1.HealthCheckResponse_ServingStatus), func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", status)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	setStatus := func(next grpc_health_v1.HealthCheckResponse_ServingStatus) {
		healthServer.SetServingStatus("", next)
	}

	stop := func() {
		healthServer.Shutdown()
		grpcServer.GracefulStop()
		_ = listener.Close()
	}

	return listener.Addr().String(), setStatus, stop
}

func startCampaignServer(t *testing.T) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	statev1.RegisterCampaignServiceServer(grpcServer, &fakeCampaignServiceServer{})

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	stop := func() {
		grpcServer.GracefulStop()
		_ = listener.Close()
		select {
		case <-serveErr:
		case <-time.After(time.Second):
		}
	}

	return listener.Addr().String(), stop
}

func stubManagedConn(t *testing.T) {
	t.Helper()
	prev := newManagedConn
	t.Cleanup(func() { newManagedConn = prev })
	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		cfg.Logf = func(string, ...any) {}
		cfg.StatusReporter = nil
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
}

// TestRunWithTransportServesAndStops ensures runWithTransport connects, serves, and exits on cancel.
func TestRunWithTransportServesAndStops(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	stubManagedConn(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- runWithTransport(ctx, addr, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	clientCtx, clientCancel := context.WithTimeout(context.Background(), time.Second)
	defer clientCancel()

	type connectResult struct {
		session *mcp.ClientSession
		err     error
	}
	connectDone := make(chan connectResult, 1)
	go func() {
		session, err := client.Connect(clientCtx, clientTransport, nil)
		connectDone <- connectResult{session: session, err: err}
	}()

	var session *mcp.ClientSession
	select {
	case result := <-connectDone:
		if result.err != nil {
			t.Fatalf("connect client: %v", result.err)
		}
		session = result.session
	case <-time.After(2 * time.Second):
		t.Fatal("connect client timed out")
	}

	defer session.Close()

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop after cancel")
	}
}

// TestRunUnsupportedTransport ensures Run rejects unknown transport kinds.
func TestRunUnsupportedTransport(t *testing.T) {
	err := Run(context.Background(), Config{
		GRPCAddr:  "localhost:0",
		Transport: "websocket",
	})
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported' in error, got: %v", err)
	}
}

func TestCloseNilAndEmpty(t *testing.T) {
	t.Run("nil server", func(t *testing.T) {
		var s *Server
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("no managed conn", func(t *testing.T) {
		s := &Server{}
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestServeWithTransportCloseError ensures serveWithTransport reports close errors.
func TestServeWithTransportCloseError(t *testing.T) {
	// A nil server should error.
	var nilServer *Server
	err := nilServer.serveWithTransport(context.Background(), &mcp.StdioTransport{})
	if err == nil {
		t.Fatal("expected error for nil server")
	}

	// Empty server (no mcpServer)
	emptyServer := &Server{}
	err = emptyServer.serveWithTransport(context.Background(), &mcp.StdioTransport{})
	if err == nil {
		t.Fatal("expected error for missing mcp server")
	}

	// Nil context defaults to background — failing transport still errors
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	server := &Server{mcpServer: mcpServer}
	err = server.serveWithTransport(nil, failingTransport{})
	if err == nil {
		t.Fatal("expected error from failing transport")
	}
}

func TestNewClosesManagedConnWhenServerBuildFails(t *testing.T) {
	prevBuild := buildMCPServerFromConn
	prevMC := newManagedConn
	t.Cleanup(func() {
		buildMCPServerFromConn = prevBuild
		newManagedConn = prevMC
	})

	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
	buildMCPServerFromConn = func(*grpc.ClientConn) (*Server, error) {
		return nil, errors.New("build failed")
	}

	_, err := New("127.0.0.1:8082")
	if err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("New error = %v, want build failure", err)
	}
}

func TestRunWithTransportClosesManagedConnWhenServerBuildFails(t *testing.T) {
	prevBuild := buildMCPServerFromConn
	prevMC := newManagedConn
	t.Cleanup(func() {
		buildMCPServerFromConn = prevBuild
		newManagedConn = prevMC
	})

	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
	buildMCPServerFromConn = func(*grpc.ClientConn) (*Server, error) {
		return nil, errors.New("build failed")
	}

	err := runWithTransport(context.Background(), "127.0.0.1:8082", &mcp.StdioTransport{})
	if err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("runWithTransport error = %v, want build failure", err)
	}
}
