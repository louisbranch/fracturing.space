package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestWaitForHealthServing(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn := dialHealthServer(t, addr)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForHealth(ctx, conn, "", nil); err != nil {
		t.Fatalf("wait for health: %v", err)
	}
}

func TestWaitForHealthTransitionsToServing(t *testing.T) {
	addr, setStatus, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	conn := dialHealthServer(t, addr)
	defer conn.Close()

	go func() {
		time.Sleep(200 * time.Millisecond)
		setStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForHealth(ctx, conn, "", nil); err != nil {
		t.Fatalf("wait for health after transition: %v", err)
	}
}

func TestWaitForHealthRespectsContext(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	conn := dialHealthServer(t, addr)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	if err := WaitForHealth(ctx, conn, "", nil); err == nil {
		t.Fatal("expected context error, got nil")
	}
}

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, func(grpc_health_v1.HealthCheckResponse_ServingStatus), func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := gogrpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", status)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	setStatus := func(next grpc_health_v1.HealthCheckResponse_ServingStatus) {
		healthServer.SetServingStatus("", next)
	}

	stop := func() {
		grpcServer.GracefulStop()
		_ = listener.Close()
		select {
		case <-serveErr:
		case <-time.After(2 * time.Second):
		}
	}

	return listener.Addr().String(), setStatus, stop
}

func dialHealthServer(t *testing.T, addr string) *gogrpc.ClientConn {
	t.Helper()

	conn, err := gogrpc.NewClient(
		addr,
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial health server: %v", err)
	}

	return conn
}
