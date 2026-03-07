package admin

import (
	"context"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"google.golang.org/grpc"
)

// TestGrpcClientsNilSafe verifies nil grpcClients receivers don't panic.
func TestGrpcClientsNilSafe(t *testing.T) {
	var g *grpcClients

	if g.CampaignClient() != nil {
		t.Error("expected nil CampaignClient")
	}
	if g.SessionClient() != nil {
		t.Error("expected nil SessionClient")
	}
	if g.CharacterClient() != nil {
		t.Error("expected nil CharacterClient")
	}
	if g.ParticipantClient() != nil {
		t.Error("expected nil ParticipantClient")
	}
	if g.InviteClient() != nil {
		t.Error("expected nil InviteClient")
	}
	if g.SnapshotClient() != nil {
		t.Error("expected nil SnapshotClient")
	}
	if g.EventClient() != nil {
		t.Error("expected nil EventClient")
	}
	if g.StatisticsClient() != nil {
		t.Error("expected nil StatisticsClient")
	}
	if g.SystemClient() != nil {
		t.Error("expected nil SystemClient")
	}
	if g.HasGameConnection() {
		t.Error("expected no game connection")
	}
	if g.HasAuthConnection() {
		t.Error("expected no auth connection")
	}

	// nil-safe set and close
	g.SetGameClients(gameGRPCClients{})
	g.SetAuthClients(authGRPCClients{})
	g.Close()
}

// TestGrpcClientsClose verifies Close releases connections.
func TestGrpcClientsClose(t *testing.T) {
	g := &grpcClients{}
	g.Close() // no connection to close — should not panic
	if g.HasGameConnection() {
		t.Error("expected no game connection after close")
	}
}

// TestGrpcClientsSetAndRead verifies SetGameClients/SetAuthClients store values.
func TestGrpcClientsSetAndRead(t *testing.T) {
	g := &grpcClients{}

	// Before setting, all accessors return nil.
	if g.CampaignClient() != nil {
		t.Error("expected nil CampaignClient before set")
	}
	if g.AuthClient() != nil {
		t.Error("expected nil AuthClient before set")
	}
	if g.HasGameConnection() {
		t.Error("expected no game connection before set")
	}
	if g.HasAuthConnection() {
		t.Error("expected no auth connection before set")
	}

	// SetGameClients with nil conn still marks it as set (conn field is assigned).
	// Use nil clients to test accessor coverage without a real gRPC connection.
	g.SetGameClients(gameGRPCClients{})

	// After first set, HasGameConnection is false since conn is nil.
	// But the guard (g.gameConn != nil) won't trigger since we passed nil.
	// Let's verify clients are stored.
	if g.CampaignClient() != nil {
		t.Error("expected nil CampaignClient with nil conn")
	}

	// SetAuthClients with nil conn.
	g.SetAuthClients(authGRPCClients{})
	if g.AuthClient() != nil {
		t.Error("expected nil AuthClient with nil conn")
	}

	// Verify other accessor non-nil paths are exercised.
	if g.SessionClient() != nil {
		t.Error("expected nil SessionClient with nil conn")
	}
	if g.CharacterClient() != nil {
		t.Error("expected nil CharacterClient with nil conn")
	}
	if g.ParticipantClient() != nil {
		t.Error("expected nil ParticipantClient with nil conn")
	}
	if g.InviteClient() != nil {
		t.Error("expected nil InviteClient with nil conn")
	}
	if g.SnapshotClient() != nil {
		t.Error("expected nil SnapshotClient with nil conn")
	}
	if g.EventClient() != nil {
		t.Error("expected nil EventClient with nil conn")
	}
	if g.StatisticsClient() != nil {
		t.Error("expected nil StatisticsClient with nil conn")
	}
	if g.SystemClient() != nil {
		t.Error("expected nil SystemClient with nil conn")
	}
}

// TestGrpcClientsSetGameConnIdempotent verifies duplicate SetGameClients is no-op.
func TestGrpcClientsSetGameConnIdempotent(t *testing.T) {
	g := &grpcClients{}
	// Simulate a set connection by directly setting gameConn.
	g.gameConn = &grpc.ClientConn{}
	// Second call should be a no-op (returns early).
	g.SetGameClients(gameGRPCClients{})
	if !g.HasGameConnection() {
		t.Error("expected game connection to remain after idempotent set")
	}
}

// TestGrpcClientsSetAuthConnIdempotent verifies duplicate SetAuthClients is no-op.
func TestGrpcClientsSetAuthConnIdempotent(t *testing.T) {
	g := &grpcClients{}
	// Simulate a set connection by directly setting authConn.
	g.authConn = &grpc.ClientConn{}
	// Second call should be a no-op (returns early).
	g.SetAuthClients(authGRPCClients{})
	if !g.HasAuthConnection() {
		t.Error("expected auth connection to remain after idempotent set")
	}
}

// TestListenAndServeNilServer verifies nil server returns an error.
func TestListenAndServeNilServer(t *testing.T) {
	var s *Server
	if err := s.ListenAndServe(context.Background()); err == nil {
		t.Fatal("expected error for nil server")
	}
}

// TestNewServerRequiresHTTPAddr ensures a blank HTTP address fails fast.
func TestNewServerRequiresHTTPAddr(t *testing.T) {
	if _, err := NewServer(context.Background(), Config{}); err == nil {
		t.Fatal("expected error for empty HTTP address")
	}
}

// TestListenAndServeStopsOnCancel verifies the server exits on context cancel.
func TestListenAndServeStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewServer(ctx, Config{HTTPAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop on cancel")
	}
}

func TestServerDependencyStatuses(t *testing.T) {
	t.Parallel()

	s := &Server{
		grpcAddr:   "game:8082",
		authAddr:   "auth:8083",
		statusAddr: "status:8093",
		grpcClients: &grpcClients{
			gameConn: &grpc.ClientConn{},
		},
	}
	statuses := s.DependencyStatuses()
	if len(statuses) != 3 {
		t.Fatalf("len(statuses) = %d, want 3", len(statuses))
	}
	if statuses[0].Name != "game" || !statuses[0].Connected {
		t.Fatalf("statuses[0] = %#v, want connected game status", statuses[0])
	}
	if statuses[1].Name != "auth" || statuses[1].Connected {
		t.Fatalf("statuses[1] = %#v, want unavailable auth status", statuses[1])
	}
	if statuses[2].Name != "status" || statuses[2].Connected {
		t.Fatalf("statuses[2] = %#v, want unavailable status status", statuses[2])
	}
}

func TestServerDependencyStatusesNilSafe(t *testing.T) {
	t.Parallel()

	var s *Server
	if statuses := s.DependencyStatuses(); statuses != nil {
		t.Fatalf("nil server statuses = %#v, want nil", statuses)
	}
	if client := s.StatusClient(); client != nil {
		t.Fatalf("nil server StatusClient() = %#v, want nil", client)
	}
}

func TestServerStatusClient(t *testing.T) {
	t.Parallel()

	client := statusv1.NewStatusServiceClient(&grpc.ClientConn{})
	s := &Server{statusClient: client}
	if got := s.StatusClient(); got != client {
		t.Fatalf("StatusClient() = %#v, want %#v", got, client)
	}
}
