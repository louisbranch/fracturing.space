package app

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
)

type loopRunnerFunc func(ctx context.Context) error

func (fn loopRunnerFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

func TestNormalizeRuntimeConfigDefaults(t *testing.T) {
	cfg, err := normalizeRuntimeConfig(RuntimeConfig{
		AuthAddr:          "auth:8083",
		GameAddr:          "game:8082",
		NotificationsAddr: "notifications:8088",
		SocialAddr:        "social:8090",
	})
	if err != nil {
		t.Fatalf("normalizeRuntimeConfig: %v", err)
	}
	if cfg.Port != defaultWorkerPort {
		t.Fatalf("port = %d, want %d", cfg.Port, defaultWorkerPort)
	}
	if cfg.DBPath != defaultWorkerDB {
		t.Fatalf("db path = %q, want %q", cfg.DBPath, defaultWorkerDB)
	}
}

func TestNewRuntimeRequiresContext(t *testing.T) {
	_, err := NewRuntime(nil, RuntimeConfig{
		AuthAddr:          "auth:8083",
		GameAddr:          "game:8082",
		NotificationsAddr: "notifications:8088",
		SocialAddr:        "social:8090",
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("NewRuntime error = %v, want context is required", err)
	}
}

func TestRunRequiresContext(t *testing.T) {
	err := Run(nil, RuntimeConfig{
		AuthAddr:          "auth:8083",
		GameAddr:          "game:8082",
		NotificationsAddr: "notifications:8088",
		SocialAddr:        "social:8090",
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Run error = %v, want context is required", err)
	}
}

func TestNormalizeRuntimeConfigRequiresAddresses(t *testing.T) {
	cases := []struct {
		name    string
		cfg     RuntimeConfig
		wantErr string
	}{
		{
			name: "missing auth",
			cfg: RuntimeConfig{
				GameAddr:          "game:8082",
				NotificationsAddr: "notifications:8088",
				SocialAddr:        "social:8090",
			},
			wantErr: "auth address is required",
		},
		{
			name: "missing game",
			cfg: RuntimeConfig{
				AuthAddr:          "auth:8083",
				NotificationsAddr: "notifications:8088",
				SocialAddr:        "social:8090",
			},
			wantErr: "game address is required",
		},
		{
			name: "missing notifications",
			cfg: RuntimeConfig{
				AuthAddr:   "auth:8083",
				GameAddr:   "game:8082",
				SocialAddr: "social:8090",
			},
			wantErr: "notifications address is required",
		},
		{
			name: "missing social",
			cfg: RuntimeConfig{
				AuthAddr:          "auth:8083",
				GameAddr:          "game:8082",
				NotificationsAddr: "notifications:8088",
			},
			wantErr: "social address is required",
		},
	}
	for _, tc := range cases {
		_, err := normalizeRuntimeConfig(tc.cfg)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("%s: err = %v, want %q", tc.name, err, tc.wantErr)
		}
	}
}

func stubManagedConn(t *testing.T) {
	t.Helper()
	previous := newManagedConn
	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		cfg.StatusReporter = nil
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
	t.Cleanup(func() {
		newManagedConn = previous
	})
}

func recordManagedConnModes(t *testing.T) *sync.Map {
	t.Helper()
	modes := &sync.Map{}
	previous := newManagedConn
	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		modes.Store(cfg.Name, cfg.Mode)
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		cfg.StatusReporter = nil
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
	t.Cleanup(func() {
		newManagedConn = previous
	})
	return modes
}

type lifecycleAuthServer struct {
	authv1.UnimplementedAuthServiceServer
}

func (lifecycleAuthServer) ListUsers(context.Context, *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	return &authv1.ListUsersResponse{}, nil
}

type lifecycleSocialServer struct {
	socialv1.UnimplementedSocialServiceServer
}

func (lifecycleSocialServer) SyncDirectoryUser(context.Context, *socialv1.SyncDirectoryUserRequest) (*socialv1.SyncDirectoryUserResponse, error) {
	return &socialv1.SyncDirectoryUserResponse{}, nil
}

func startLifecycleDependencyServers(t *testing.T) (string, string, string, string) {
	t.Helper()

	startServer := func(register func(*grpc.Server), label string) string {
		t.Helper()
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("%s listen: %v", label, err)
		}
		server := grpc.NewServer()
		register(server)
		go func() {
			_ = server.Serve(listener)
		}()
		t.Cleanup(func() {
			server.Stop()
			_ = listener.Close()
		})
		return listener.Addr().String()
	}

	authAddr := startServer(func(server *grpc.Server) {
		authv1.RegisterAuthServiceServer(server, lifecycleAuthServer{})
	}, "auth")
	socialAddr := startServer(func(server *grpc.Server) {
		socialv1.RegisterSocialServiceServer(server, lifecycleSocialServer{})
	}, "social")
	gameAddr := startServer(func(server *grpc.Server) {}, "game")
	notificationsAddr := startServer(func(server *grpc.Server) {}, "notifications")
	return authAddr, gameAddr, notificationsAddr, socialAddr
}

func TestNewRuntimeBuildsAndCloses(t *testing.T) {
	stubManagedConn(t)
	authAddr, gameAddr, notificationsAddr, socialAddr := startLifecycleDependencyServers(t)

	srv, err := NewRuntime(context.Background(), RuntimeConfig{
		Port:              freeWorkerTCPPort(t),
		AuthAddr:          authAddr,
		GameAddr:          gameAddr,
		NotificationsAddr: notificationsAddr,
		SocialAddr:        socialAddr,
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if got := srv.Addr(); got == "" {
		t.Fatal("expected runtime listener address")
	}
	srv.Close()
	srv.Close()
}

func TestNewRuntime_UsesOptionalManagedConnsForGameAndNotifications(t *testing.T) {
	modes := recordManagedConnModes(t)
	authAddr, _, _, socialAddr := startLifecycleDependencyServers(t)

	srv, err := NewRuntime(context.Background(), RuntimeConfig{
		Port:              freeWorkerTCPPort(t),
		AuthAddr:          authAddr,
		GameAddr:          "127.0.0.1:1",
		NotificationsAddr: "127.0.0.1:2",
		SocialAddr:        socialAddr,
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	t.Cleanup(srv.Close)

	assertManagedConnMode(t, modes, "auth", platformgrpc.ModeRequired)
	assertManagedConnMode(t, modes, "social", platformgrpc.ModeRequired)
	assertManagedConnMode(t, modes, "game", platformgrpc.ModeOptional)
	assertManagedConnMode(t, modes, "notifications", platformgrpc.ModeOptional)
}

func TestRuntimeServeStopsOnContextCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	loopStarted := make(chan struct{}, 1)
	runtime := &Runtime{
		listener:   listener,
		grpcServer: grpc.NewServer(),
		health:     health.NewServer(),
		loop: loopRunnerFunc(func(ctx context.Context) error {
			loopStarted <- struct{}{}
			<-ctx.Done()
			return nil
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(ctx)
	}()

	select {
	case <-loopStarted:
	case <-time.After(time.Second):
		t.Fatal("loop did not start")
	}

	cancel()
	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("Serve: %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop after cancellation")
	}
}

func TestRuntimeServeRequiresContext(t *testing.T) {
	stubManagedConn(t)
	authAddr, gameAddr, notificationsAddr, socialAddr := startLifecycleDependencyServers(t)

	runtime, err := NewRuntime(context.Background(), RuntimeConfig{
		Port:              freeWorkerTCPPort(t),
		AuthAddr:          authAddr,
		GameAddr:          gameAddr,
		NotificationsAddr: notificationsAddr,
		SocialAddr:        socialAddr,
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	t.Cleanup(func() {
		runtime.Close()
	})

	if err := runtime.Serve(nil); err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Serve error = %v, want context is required", err)
	}
}

func assertManagedConnMode(t *testing.T, modes *sync.Map, name string, want platformgrpc.ManagedConnMode) {
	t.Helper()
	gotRaw, ok := modes.Load(name)
	if !ok {
		t.Fatalf("missing managed conn mode record for %s", name)
	}
	got, ok := gotRaw.(platformgrpc.ManagedConnMode)
	if !ok {
		t.Fatalf("managed conn mode type for %s = %T", name, gotRaw)
	}
	if got != want {
		t.Fatalf("managed conn mode for %s = %v, want %v", name, got, want)
	}
}

func TestRuntimeServeReturnsLoopErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	runtime := &Runtime{
		listener:   listener,
		grpcServer: grpc.NewServer(),
		health:     health.NewServer(),
		loop: loopRunnerFunc(func(context.Context) error {
			return errors.New("loop failure")
		}),
	}

	err = runtime.Serve(context.Background())
	if err == nil || !strings.Contains(err.Error(), "run worker loop: loop failure") {
		t.Fatalf("Serve error = %v, want loop failure", err)
	}
}

func TestRuntimeServeRequiresConfiguredRuntime(t *testing.T) {
	err := (&Runtime{}).Serve(context.Background())
	if err == nil || !strings.Contains(err.Error(), "runtime is not configured") {
		t.Fatalf("Serve error = %v, want runtime configuration error", err)
	}
}

func freeWorkerTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", listener.Addr())
	}
	return addr.Port
}
