package admin

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

func TestDialGameGRPCDialError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, _, _, _, _, _, _, _, _, _, _, err := dialGameGRPC(ctx, Config{
		GRPCAddr:        "127.0.0.1:1",
		GRPCDialTimeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDialGameGRPCHealthError(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, _, _, _, _, _, _, _, _, _, _, err := dialGameGRPC(ctx, Config{
		GRPCAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "admin game gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCHealthError(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "admin auth gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialGameGRPCSuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, _, _, campaignClient, _, _, _, _, _, _, _, _, err := dialGameGRPC(ctx, Config{
		GRPCAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection")
	}
	if campaignClient == nil {
		t.Fatal("expected campaign client")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func TestDialAuthGRPCSuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, client, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
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

func TestConnectGameGRPCWithRetryNoAddr(t *testing.T) {
	connectGameGRPCWithRetry(context.Background(), Config{}, &grpcClients{})
}

func TestConnectGameGRPCWithRetryCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connectGameGRPCWithRetry(ctx, Config{GRPCAddr: "127.0.0.1:1"}, &grpcClients{})
}

func TestConnectGameGRPCWithRetrySuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clients := &grpcClients{}
	connectGameGRPCWithRetry(ctx, Config{
		GRPCAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	}, clients)

	if !clients.HasGameConnection() {
		t.Fatal("expected game connection")
	}
	clients.Close()
}

func TestConnectAuthGRPCWithRetryNoAddr(t *testing.T) {
	connectAuthGRPCWithRetry(context.Background(), Config{}, &grpcClients{})
}

func TestConnectAuthGRPCWithRetryCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connectAuthGRPCWithRetry(ctx, Config{AuthAddr: "127.0.0.1:1"}, &grpcClients{})
}

func TestConnectAuthGRPCWithRetrySuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clients := &grpcClients{}
	connectAuthGRPCWithRetry(ctx, Config{
		AuthAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	}, clients)

	if !clients.HasAuthConnection() {
		t.Fatal("expected auth connection")
	}
	clients.Close()
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
