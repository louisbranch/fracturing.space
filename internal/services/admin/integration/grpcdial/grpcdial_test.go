package grpcdial

import (
	"context"
	"errors"
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

func TestDialReturnsZeroClientsForEmptyAddress(t *testing.T) {
	t.Parallel()

	gameClients, err := DialGame(context.Background(), "", time.Second, "admin_dashboard")
	if err != nil {
		t.Fatalf("DialGame: %v", err)
	}
	if gameClients.Conn != nil {
		t.Fatalf("DialGame Conn = %v, want nil", gameClients.Conn)
	}

	authClients, err := DialAuth(context.Background(), "  ", time.Second)
	if err != nil {
		t.Fatalf("DialAuth: %v", err)
	}
	if authClients.Conn != nil {
		t.Fatalf("DialAuth Conn = %v, want nil", authClients.Conn)
	}
}

func TestDialGameHealthError(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := DialGame(ctx, addr, 100*time.Millisecond, "admin_dashboard")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "admin game gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthHealthError(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := DialAuth(ctx, addr, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "admin auth gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialGameAddsAdminOverrideMetadata(t *testing.T) {
	addr, campaignServer, stop := startCampaignHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clients, err := DialGame(ctx, addr, 200*time.Millisecond, "admin_dashboard")
	if err != nil {
		t.Fatalf("DialGame: %v", err)
	}
	defer clients.Conn.Close()

	_, err = clients.CampaignClient.ListCampaigns(context.Background(), &statev1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	if got := campaignServer.lastMetadata.Get("x-fracturing-space-platform-role"); len(got) != 1 || got[0] != "ADMIN" {
		t.Fatalf("platform role metadata = %v, want [ADMIN]", got)
	}
	if got := campaignServer.lastMetadata.Get("x-fracturing-space-authz-override-reason"); len(got) != 1 || got[0] != "admin_dashboard" {
		t.Fatalf("override reason metadata = %v, want [admin_dashboard]", got)
	}
}

func TestConnectWithRetryNoopsOnNilCallbacks(t *testing.T) {
	t.Parallel()

	ConnectWithRetry(context.Background(), "127.0.0.1:1", nil, nil, "ok %s", "err %v")
}

func TestConnectWithRetryReturnsWhenAlreadyConnected(t *testing.T) {
	t.Parallel()

	attempts := 0
	ConnectWithRetry(
		context.Background(),
		"127.0.0.1:1",
		func() bool { return true },
		func(context.Context) error {
			attempts++
			return nil
		},
		"ok %s",
		"err %v",
	)

	if attempts != 0 {
		t.Fatalf("connect attempts = %d, want 0", attempts)
	}
}

func TestConnectWithRetryStopsOnCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	ConnectWithRetry(
		ctx,
		"127.0.0.1:1",
		func() bool { return false },
		func(context.Context) error {
			attempts++
			return errors.New("dial should not run")
		},
		"ok %s",
		"err %v",
	)

	if attempts != 0 {
		t.Fatalf("connect attempts = %d, want 0", attempts)
	}
}

func TestConnectWithRetryRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	connected := false
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ConnectWithRetry(
		ctx,
		"127.0.0.1:1",
		func() bool { return connected },
		func(context.Context) error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			connected = true
			return nil
		},
		"ok %s",
		"err %v",
	)

	if !connected {
		t.Fatal("expected connection to succeed")
	}
	if attempts != 2 {
		t.Fatalf("connect attempts = %d, want 2", attempts)
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
