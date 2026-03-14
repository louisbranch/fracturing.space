package testkit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	authserver "github.com/louisbranch/fracturing.space/internal/services/auth/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// SetGameDBPaths points the game runtime databases at files inside base.
func SetGameDBPaths(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", filepath.Join(base, "game-events.db")); err != nil {
		t.Fatalf("set game events db path: %v", err)
	}
	if err := setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", filepath.Join(base, "game-projections.db")); err != nil {
		t.Fatalf("set game projections db path: %v", err)
	}
	if err := setenv("FRACTURING_SPACE_GAME_CONTENT_DB_PATH", filepath.Join(base, "game-content.db")); err != nil {
		t.Fatalf("set game content db path: %v", err)
	}
	if err := setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key"); err != nil {
		t.Fatalf("set game event hmac key: %v", err)
	}
}

// SetTempGameDBPaths points the game runtime databases at per-test temp files.
func SetTempGameDBPaths(t *testing.T) {
	t.Helper()

	SetGameDBPaths(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// SetAuthDBPath points the auth runtime database at a file inside base.
func SetAuthDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(base, "auth.db")); err != nil {
		t.Fatalf("set auth db path: %v", err)
	}
}

// SetTempAuthDBPath points the auth runtime database at a per-test temp file.
func SetTempAuthDBPath(t *testing.T) {
	t.Helper()

	SetAuthDBPath(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// StartAuthServer boots the auth server for runtime tests and waits for readiness.
func StartAuthServer(t *testing.T) (string, func()) {
	t.Helper()

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
	WaitForGRPCHealth(t, authAddr)
	stop := func() {
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

	return authAddr, stop
}

// WaitForGRPCHealth waits for a gRPC server to report SERVING on the default service.
func WaitForGRPCHealth(t *testing.T, addr string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC server %q: %v", addr, err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	backoff := 100 * time.Millisecond
	for {
		callCtx, callCancel := context.WithTimeout(ctx, time.Second)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		callCancel()
		if err == nil && response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return
		}

		select {
		case <-ctx.Done():
			if err != nil {
				t.Fatalf("wait for gRPC health at %q: %v", addr, err)
			}
			t.Fatalf("wait for gRPC health at %q: %v", addr, ctx.Err())
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, time.Second)
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
