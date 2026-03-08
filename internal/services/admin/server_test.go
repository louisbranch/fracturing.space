package admin

import (
	"context"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"google.golang.org/grpc"
)

// TestServerNilSafeAccessors verifies nil server receivers don't panic.
func TestServerNilSafeAccessors(t *testing.T) {
	var s *Server

	if s.CampaignClient() != nil {
		t.Error("expected nil CampaignClient")
	}
	if s.SessionClient() != nil {
		t.Error("expected nil SessionClient")
	}
	if s.CharacterClient() != nil {
		t.Error("expected nil CharacterClient")
	}
	if s.ParticipantClient() != nil {
		t.Error("expected nil ParticipantClient")
	}
	if s.InviteClient() != nil {
		t.Error("expected nil InviteClient")
	}
	if s.SnapshotClient() != nil {
		t.Error("expected nil SnapshotClient")
	}
	if s.EventClient() != nil {
		t.Error("expected nil EventClient")
	}
	if s.StatisticsClient() != nil {
		t.Error("expected nil StatisticsClient")
	}
	if s.SystemClient() != nil {
		t.Error("expected nil SystemClient")
	}
	if s.AuthClient() != nil {
		t.Error("expected nil AuthClient")
	}
	if s.AccountClient() != nil {
		t.Error("expected nil AccountClient")
	}
	if s.DaggerheartContentClient() != nil {
		t.Error("expected nil DaggerheartContentClient")
	}

	// nil-safe close
	s.Close()
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

func TestServerStatusClientNilSafe(t *testing.T) {
	t.Parallel()

	var s *Server
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
