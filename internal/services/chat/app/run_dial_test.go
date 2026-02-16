package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestRunReturnsInitErrorForInvalidConfig(t *testing.T) {
	err := Run(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !strings.Contains(err.Error(), "init chat server") {
		t.Fatalf("error = %v, want init chat server prefix", err)
	}
}

func TestRunStartsAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, Config{HTTPAddr: "127.0.0.1:0"})
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop on cancel")
	}
}

func TestDialGameGRPCEmptyAddr(t *testing.T) {
	conn, client, err := dialGameGRPC(context.Background(), Config{})
	if err != nil {
		t.Fatalf("dialGameGRPC returned error: %v", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn when game addr is empty")
	}
	if client != nil {
		t.Fatal("expected nil participant client when game addr is empty")
	}
}

func TestDialGameGRPCReturnsErrorWhenUnreachable(t *testing.T) {
	conn, client, err := dialGameGRPC(context.Background(), Config{
		GameAddr:        "127.0.0.1:1",
		GRPCDialTimeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected dial error")
	}
	if conn != nil {
		t.Fatal("expected nil conn on dial error")
	}
	if client != nil {
		t.Fatal("expected nil client on dial error")
	}
}

func TestDialGameGRPCSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		select {
		case <-serveErr:
		default:
		}
	})

	conn, client, err := dialGameGRPC(context.Background(), Config{
		GameAddr:        listener.Addr().String(),
		GRPCDialTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("dialGameGRPC: %v", err)
	}
	if conn == nil {
		t.Fatal("expected conn")
	}
	if client == nil {
		t.Fatal("expected participant client")
	}
	if closeErr := conn.Close(); closeErr != nil {
		t.Fatalf("close conn: %v", closeErr)
	}
}
