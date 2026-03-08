package app

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
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
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
		DBPath:            filepath.Join(t.TempDir(), "worker.db"),
	})
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("NewRuntime error = %v, want context is required", err)
	}
}

func TestRunRequiresContext(t *testing.T) {
	err := Run(nil, RuntimeConfig{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
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
				SocialAddr:        "social:8090",
				NotificationsAddr: "notifications:8088",
			},
			wantErr: "auth address is required",
		},
		{
			name: "missing notifications",
			cfg: RuntimeConfig{
				AuthAddr:   "auth:8083",
				SocialAddr: "social:8090",
			},
			wantErr: "notifications address is required",
		},
		{
			name: "missing social",
			cfg: RuntimeConfig{
				AuthAddr:          "auth:8083",
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

func TestNewRuntimeBuildsAndCloses(t *testing.T) {
	stubManagedConn(t)

	srv, err := NewRuntime(context.Background(), RuntimeConfig{
		Port:              freeWorkerTCPPort(t),
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
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

	runtime, err := NewRuntime(context.Background(), RuntimeConfig{
		Port:              freeWorkerTCPPort(t),
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
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
