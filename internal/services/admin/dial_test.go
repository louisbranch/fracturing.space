package admin

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestDialGameGRPCDialError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := dialGameGRPC(ctx, Config{
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

	_, err := dialGameGRPC(ctx, Config{
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

	_, err := dialAuthGRPC(ctx, Config{
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

	clients, err := dialGameGRPC(ctx, Config{
		GRPCAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	if clients.conn == nil {
		t.Fatal("expected connection")
	}
	if clients.campaignClient == nil {
		t.Fatal("expected campaign client")
	}
	if err := clients.conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func TestDialAuthGRPCSuccess(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clients, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        addr,
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dial auth gRPC: %v", err)
	}
	if clients.conn == nil {
		t.Fatal("expected connection")
	}
	if clients.authClient == nil {
		t.Fatal("expected auth client")
	}
	if err := clients.conn.Close(); err != nil {
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

func TestDialGameGRPCAddsAdminOverrideMetadata(t *testing.T) {
	addr, campaignServer, stop := startCampaignHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clients, err := dialGameGRPC(ctx, Config{
		GRPCAddr:        addr,
		GRPCDialTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	defer clients.conn.Close()

	_, err = clients.campaignClient.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if got := campaignServer.lastMetadata.Get("x-fracturing-space-platform-role"); len(got) != 1 || got[0] != "ADMIN" {
		t.Fatalf("platform role metadata = %v, want [ADMIN]", got)
	}
	if got := campaignServer.lastMetadata.Get("x-fracturing-space-authz-override-reason"); len(got) != 1 || got[0] != "admin_dashboard" {
		t.Fatalf("override reason metadata = %v, want [admin_dashboard]", got)
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

type metadataCaptureCampaignServer struct {
	statev1.UnimplementedCampaignServiceServer
	lastMetadata metadata.MD
}

func (s *metadataCaptureCampaignServer) ListCampaigns(ctx context.Context, _ *statev1.ListCampaignsRequest) (*statev1.ListCampaignsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.lastMetadata = md.Copy()
	return &statev1.ListCampaignsResponse{}, nil
}

func startCampaignHealthServer(
	t *testing.T,
	healthStatus grpc_health_v1.HealthCheckResponse_ServingStatus,
) (string, *metadataCaptureCampaignServer, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := gogrpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthStatus)

	campaignServer := &metadataCaptureCampaignServer{}
	statev1.RegisterCampaignServiceServer(grpcServer, campaignServer)

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

	return listener.Addr().String(), campaignServer, stop
}

func TestGameGRPCCallContextOmitsAdminOverride(t *testing.T) {
	// Admin override is injected by connection-level interceptors, not per-call.
	// gameGRPCCallContext must NOT add admin override to outgoing metadata.
	h := &Handler{}
	ctx, cancel := h.gameGRPCCallContext(context.Background())
	defer cancel()

	md, _ := metadata.FromOutgoingContext(ctx)
	if got := md.Get("x-fracturing-space-platform-role"); len(got) != 0 {
		t.Errorf("expected no platform role in per-call metadata, got %v", got)
	}
	if got := md.Get("x-fracturing-space-authz-override-reason"); len(got) != 0 {
		t.Errorf("expected no override reason in per-call metadata, got %v", got)
	}
}
