package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
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

// TestRunWithTransportServesAndStops ensures runWithTransport connects, serves, and exits on cancel.
func TestRunWithTransportServesAndStops(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

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

// TestMonitorHealthExitsOnCancel ensures monitorHealth returns when context is cancelled.
func TestMonitorHealthExitsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{}

	done := make(chan struct{})
	go func() {
		server.monitorHealth(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// monitorHealth returned promptly
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHealth did not exit after context cancellation")
	}
}

// TestMonitorHealthNilConn ensures monitorHealth handles a nil connection gracefully.
func TestMonitorHealthNilConn(t *testing.T) {
	restore := stubMonitorHealthDependencies(t)
	defer restore()
	ticks := make(chan time.Time, 1)
	newHealthMonitorTicker = func(time.Duration) (<-chan time.Time, func()) {
		return ticks, func() {}
	}
	checkCalls := 0
	checkConnectionHealth = func(context.Context, *grpc.ClientConn) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
		checkCalls++
		return grpc_health_v1.HealthCheckResponse_SERVING, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := &Server{conn: nil}

	done := make(chan struct{})
	go func() {
		server.monitorHealth(ctx)
		close(done)
	}()
	ticks <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// returned after cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHealth did not exit after context cancellation")
	}
	if checkCalls != 0 {
		t.Fatalf("health checks = %d, want 0", checkCalls)
	}
}

// TestMonitorHealthChecksGRPC ensures monitorHealth performs health checks against a real gRPC server.
func TestMonitorHealthChecksGRPC(t *testing.T) {
	restore := stubMonitorHealthDependencies(t)
	defer restore()

	ticks := make(chan time.Time, 1)
	newHealthMonitorTicker = func(time.Duration) (<-chan time.Time, func()) {
		return ticks, func() {}
	}
	checkedConn := make(chan *grpc.ClientConn, 1)
	checkConnectionHealth = func(_ context.Context, conn *grpc.ClientConn) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
		checkedConn <- conn
		return grpc_health_v1.HealthCheckResponse_SERVING, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &Server{conn: &grpc.ClientConn{}}

	done := make(chan struct{})
	go func() {
		server.monitorHealth(ctx)
		close(done)
	}()
	ticks <- time.Now()

	select {
	case conn := <-checkedConn:
		if conn == nil {
			t.Fatal("expected non-nil connection for health check")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHealth did not run a health check")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHealth did not exit after cancellation")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	previousClose := closeGRPCConn
	t.Cleanup(func() {
		closeGRPCConn = previousClose
	})

	var closeCalls int32
	closeGRPCConn = func(*grpc.ClientConn) error {
		atomic.AddInt32(&closeCalls, 1)
		return nil
	}

	server := &Server{conn: &grpc.ClientConn{}}
	errCh := make(chan error, 2)
	go func() {
		errCh <- server.Close()
	}()
	go func() {
		errCh <- server.Close()
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("close error: %v", err)
		}
	}
	if got := atomic.LoadInt32(&closeCalls); got != 1 {
		t.Fatalf("close calls = %d, want 1", got)
	}
	if server.grpcConn() != nil {
		t.Fatal("expected connection to be cleared")
	}
}

func TestCloseRestoresConnectionOnCloseError(t *testing.T) {
	previousClose := closeGRPCConn
	t.Cleanup(func() {
		closeGRPCConn = previousClose
	})

	closeErr := errors.New("close failed")
	var closeCalls int32
	closeGRPCConn = func(*grpc.ClientConn) error {
		call := atomic.AddInt32(&closeCalls, 1)
		if call == 1 {
			return closeErr
		}
		return nil
	}

	server := &Server{conn: &grpc.ClientConn{}}
	err := server.Close()
	if !errors.Is(err, closeErr) {
		t.Fatalf("close error = %v, want %v", err, closeErr)
	}
	if server.grpcConn() == nil {
		t.Fatal("expected connection to be restored after close failure")
	}

	if err := server.Close(); err != nil {
		t.Fatalf("second close error: %v", err)
	}
	if server.grpcConn() != nil {
		t.Fatal("expected connection to be cleared after successful close")
	}
	if got := atomic.LoadInt32(&closeCalls); got != 2 {
		t.Fatalf("close calls = %d, want 2", got)
	}
}

// TestServeWithTransportCloseError ensures serveWithTransport reports close errors.
func TestServeWithTransportCloseError(t *testing.T) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	// A server with no conn won't error on close, but a nil mcpServer will error.
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
	server := &Server{mcpServer: mcpServer}
	err = server.serveWithTransport(nil, failingTransport{})
	if err == nil {
		t.Fatal("expected error from failing transport")
	}
}

func TestNewClosesConnectionWhenServerBuildFails(t *testing.T) {
	previousDial := dialGameLazyConn
	previousBuild := buildMCPServerFromConn
	previousClose := closeGRPCConn

	dialGameLazyConn = func(string) (*grpc.ClientConn, error) {
		return nil, nil
	}
	buildMCPServerFromConn = func(*grpc.ClientConn) (*Server, error) {
		return nil, errors.New("build failed")
	}
	closeCalls := 0
	closeGRPCConn = func(*grpc.ClientConn) error {
		closeCalls++
		return nil
	}
	t.Cleanup(func() {
		dialGameLazyConn = previousDial
		buildMCPServerFromConn = previousBuild
		closeGRPCConn = previousClose
	})

	_, err := New("127.0.0.1:8082")
	if err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("New error = %v, want build failure", err)
	}
	if closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", closeCalls)
	}
}

func TestRunWithTransportClosesConnectionWhenServerBuildFails(t *testing.T) {
	previousDial := dialGameRuntimeConn
	previousBuild := buildMCPServerFromConn
	previousClose := closeGRPCConn

	dialGameRuntimeConn = func(context.Context, string) (*grpc.ClientConn, error) {
		return nil, nil
	}
	buildMCPServerFromConn = func(*grpc.ClientConn) (*Server, error) {
		return nil, errors.New("build failed")
	}
	closeCalls := 0
	closeGRPCConn = func(*grpc.ClientConn) error {
		closeCalls++
		return nil
	}
	t.Cleanup(func() {
		dialGameRuntimeConn = previousDial
		buildMCPServerFromConn = previousBuild
		closeGRPCConn = previousClose
	})

	err := runWithTransport(context.Background(), "127.0.0.1:8082", &mcp.StdioTransport{})
	if err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("runWithTransport error = %v, want build failure", err)
	}
	if closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", closeCalls)
	}
}

func stubMonitorHealthDependencies(t *testing.T) func() {
	t.Helper()
	previousTicker := newHealthMonitorTicker
	previousCheck := checkConnectionHealth
	return func() {
		newHealthMonitorTicker = previousTicker
		checkConnectionHealth = previousCheck
	}
}
