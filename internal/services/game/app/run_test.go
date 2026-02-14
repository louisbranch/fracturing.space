package server

import (
	"context"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	authserver "github.com/louisbranch/fracturing.space/internal/services/auth/app"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

// TestServeStopsOnContext verifies the server serves and stops on cancel.
func TestServeStopsOnContext(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := normalizeAddress(t, grpcServer.listener.Addr().String())

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	client := pb.NewDaggerheartServiceClient(conn)
	callCtx, callCancel := context.WithTimeout(context.Background(), time.Second)
	defer callCancel()
	if _, err := client.ActionRoll(callCtx, &pb.ActionRollRequest{}); err != nil {
		t.Fatalf("action roll: %v", err)
	}

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

// TestHealthCheckReportsServing ensures gRPC health checks report SERVING.
func TestHealthCheckReportsServing(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := normalizeAddress(t, grpcServer.listener.Addr().String())
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	services := []string{"", "systems.daggerheart.v1.DaggerheartService", "game.v1.CampaignService"}
	for _, service := range services {
		callCtx, callCancel := context.WithTimeout(context.Background(), time.Second)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: service})
		callCancel()
		if err != nil {
			t.Fatalf("health check %q: %v", service, err)
		}
		if response.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Fatalf("health check %q = %v, want SERVING", service, response.GetStatus())
		}
	}

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

// TestServerMetadataHeaders ensures request metadata is echoed in headers.
func TestServerMetadataHeaders(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := normalizeAddress(t, grpcServer.listener.Addr().String())
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	client := pb.NewDaggerheartServiceClient(conn)

	requestID := "req-123"
	invocationID := "inv-456"
	md := metadata.Pairs(
		grpcmeta.RequestIDHeader, requestID,
		grpcmeta.InvocationIDHeader, invocationID,
	)
	callCtx := metadata.NewOutgoingContext(context.Background(), md)

	var header metadata.MD
	callTimeout, callCancel := context.WithTimeout(callCtx, time.Second)
	_, err = client.ActionRoll(callTimeout, &pb.ActionRollRequest{}, grpc.Header(&header))
	callCancel()
	if err != nil {
		t.Fatalf("action roll: %v", err)
	}

	responseRequestID := header.Get(grpcmeta.RequestIDHeader)
	if len(responseRequestID) != 1 || responseRequestID[0] != requestID {
		t.Fatalf("request id header = %v, want %q", responseRequestID, requestID)
	}
	responseInvocationID := header.Get(grpcmeta.InvocationIDHeader)
	if len(responseInvocationID) != 1 || responseInvocationID[0] != invocationID {
		t.Fatalf("invocation id header = %v, want %q", responseInvocationID, invocationID)
	}

	callTimeout, callCancel = context.WithTimeout(context.Background(), time.Second)
	var generatedHeader metadata.MD
	_, err = client.ActionRoll(callTimeout, &pb.ActionRollRequest{}, grpc.Header(&generatedHeader))
	callCancel()
	if err != nil {
		t.Fatalf("action roll without request id: %v", err)
	}
	if len(generatedHeader.Get(grpcmeta.RequestIDHeader)) != 1 {
		t.Fatal("expected generated request id header")
	}

	cancel()
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

// TestRunPortInUse verifies Run returns an error when the port is occupied.
func TestRunPortInUse(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split address %q: %v", listener.Addr().String(), err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port %q: %v", port, err)
	}

	if err := Run(context.Background(), portNumber); err == nil {
		t.Fatal("expected error when port is already in use")
	}
}

// TestServeReturnsOnCancel verifies Serve returns promptly on cancel without connections.
func TestServeReturnsOnCancel(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop on cancel")
	}
}

// TestServeReturnsErrorOnClosedListener verifies Serve reports listener errors.
func TestServeReturnsErrorOnClosedListener(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()
	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := grpcServer.listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := grpcServer.Serve(ctx); err == nil {
		t.Fatal("expected serve error after closing listener")
	}
}

func TestRunWithAddrInvalid(t *testing.T) {
	setTempDBPath(t)
	stopAuth := startAuthServer(t)
	defer stopAuth()

	if err := RunWithAddr(context.Background(), "invalid::addr"); err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func normalizeAddress(t *testing.T, addr string) string {
	t.Helper()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split address %q: %v", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func setTempDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", filepath.Join(base, "game-events.db"))
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", filepath.Join(base, "game-projections.db"))
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
}

func setTempAuthDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(base, "auth.db"))
}

func startAuthServer(t *testing.T) func() {
	t.Helper()

	setTempAuthDBPath(t)
	ctx, cancel := context.WithCancel(context.Background())
	authServer, err := authserver.New(0, "")
	if err != nil {
		cancel()
		t.Fatalf("new auth server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- authServer.Serve(ctx)
	}()

	authAddr := authServer.Addr()
	waitForGRPCHealth(t, authAddr)
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", authAddr)

	return func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("auth server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for auth server to stop")
		}
	}
}

func waitForGRPCHealth(t *testing.T, addr string) {
	t.Helper()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := platformgrpc.WaitForHealth(ctx, conn, "", nil); err != nil {
		t.Fatalf("wait for gRPC health: %v", err)
	}
}
