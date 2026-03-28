package testkit

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	authserver "github.com/louisbranch/fracturing.space/internal/services/auth/app"
	discoveryserver "github.com/louisbranch/fracturing.space/internal/services/discovery/app"
	notificationsserver "github.com/louisbranch/fracturing.space/internal/services/notifications/app"
	socialserver "github.com/louisbranch/fracturing.space/internal/services/social/app"
	userhubapp "github.com/louisbranch/fracturing.space/internal/services/userhub/app"
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

// SetSocialDBPath points the social runtime database at a file inside base.
func SetSocialDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(base, "social.db")); err != nil {
		t.Fatalf("set social db path: %v", err)
	}
}

// SetTempSocialDBPath points the social runtime database at a per-test temp file.
func SetTempSocialDBPath(t *testing.T) {
	t.Helper()

	SetSocialDBPath(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// SetNotificationsDBPath points the notifications runtime database at a file inside base.
func SetNotificationsDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_NOTIFICATIONS_DB_PATH", filepath.Join(base, "notifications.db")); err != nil {
		t.Fatalf("set notifications db path: %v", err)
	}
}

// SetTempNotificationsDBPath points the notifications runtime database at a per-test temp file.
func SetTempNotificationsDBPath(t *testing.T) {
	t.Helper()

	SetNotificationsDBPath(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// SetInviteDBPath points the invite runtime database at a file inside base.
func SetInviteDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_INVITE_DB_PATH", filepath.Join(base, "invite.db")); err != nil {
		t.Fatalf("set invite db path: %v", err)
	}
}

// SetTempInviteDBPath points the invite runtime database at a per-test temp file.
func SetTempInviteDBPath(t *testing.T) {
	t.Helper()

	SetInviteDBPath(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// SetWorkerDBPath points the worker runtime database at a file inside base.
func SetWorkerDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_WORKER_DB_PATH", filepath.Join(base, "worker.db")); err != nil {
		t.Fatalf("set worker db path: %v", err)
	}
}

// SetTempWorkerDBPath points the worker runtime database at a per-test temp file.
func SetTempWorkerDBPath(t *testing.T) {
	t.Helper()

	SetWorkerDBPath(t, t.TempDir(), func(key, value string) error {
		t.Setenv(key, value)
		return nil
	})
}

// SetDiscoveryDBPath points the discovery runtime database at a file inside base.
func SetDiscoveryDBPath(t *testing.T, base string, setenv func(string, string) error) {
	t.Helper()

	if setenv == nil {
		t.Fatal("setenv function is required")
	}
	if err := setenv("FRACTURING_SPACE_DISCOVERY_DB_PATH", filepath.Join(base, "discovery.db")); err != nil {
		t.Fatalf("set discovery db path: %v", err)
	}
}

// SetTempDiscoveryDBPath points the discovery runtime database at a per-test temp file.
func SetTempDiscoveryDBPath(t *testing.T) {
	t.Helper()

	SetDiscoveryDBPath(t, t.TempDir(), func(key, value string) error {
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

// StartSocialServer boots the social server for runtime tests and waits for readiness.
func StartSocialServer(t *testing.T) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := socialserver.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("new social server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx)
	}()

	addr := srv.Addr()
	WaitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("social server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for social server to stop")
		}
	}

	return addr, stop
}

// StartNotificationsServer boots the notifications server for runtime tests and waits for readiness.
func StartNotificationsServer(t *testing.T) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := notificationsserver.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("new notifications server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx)
	}()

	addr := srv.Addr()
	WaitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("notifications server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for notifications server to stop")
		}
	}

	return addr, stop
}

// StartUserHubServer boots the userhub server for runtime tests and waits for readiness.
func StartUserHubServer(t *testing.T, cfg userhubapp.RuntimeConfig) (string, func()) {
	t.Helper()

	if cfg.Port <= 0 {
		cfg.Port = unusedTCPPort(t)
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := userhubapp.New(ctx, cfg)
	if err != nil {
		cancel()
		t.Fatalf("new userhub server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx)
	}()

	addr := srv.Addr()
	WaitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("userhub server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for userhub server to stop")
		}
	}

	return addr, stop
}

// StartDiscoveryServer boots the discovery server for runtime tests and waits for readiness.
func StartDiscoveryServer(t *testing.T, gameAddr string) (string, func()) {
	t.Helper()

	t.Setenv("FRACTURING_SPACE_GAME_ADDR", gameAddr)

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := discoveryserver.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("new discovery server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx)
	}()

	addr := srv.Addr()
	WaitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("discovery server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for discovery server to stop")
		}
	}

	return addr, stop
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

func unusedTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate tcp port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", listener.Addr())
	}
	return addr.Port
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
