package server

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDialAuthGRPCDialError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, "127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dial auth gRPC") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCHealthError(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, addr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCSuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, client, err := dialAuthGRPC(ctx, addr)
	if err != nil {
		t.Fatalf("dial auth gRPC: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection")
	}
	if client == nil {
		t.Fatal("expected auth client")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, func()) {
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
