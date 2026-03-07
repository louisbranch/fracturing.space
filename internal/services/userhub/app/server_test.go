package app

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func validRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		GameAddr:          "game:8082",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
	}
}

func TestRunRequiresGameAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
	})
	if err == nil || !strings.Contains(err.Error(), "game address is required") {
		t.Fatalf("Run error = %v, want game address validation", err)
	}
}

func TestNewRequiresContext(t *testing.T) {
	t.Parallel()

	_, err := New(nil, validRuntimeConfig())
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("New error = %v, want context is required", err)
	}
}

func TestRunRequiresContext(t *testing.T) {
	t.Parallel()

	err := Run(nil, validRuntimeConfig())
	if err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Run error = %v, want context is required", err)
	}
}

func TestRunRequiresSocialAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		GameAddr:          "game:8082",
		NotificationsAddr: "notifications:8088",
	})
	if err == nil || !strings.Contains(err.Error(), "social address is required") {
		t.Fatalf("Run error = %v, want social address validation", err)
	}
}

func TestRunRequiresNotificationsAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		GameAddr:   "game:8082",
		SocialAddr: "social:8090",
	})
	if err == nil || !strings.Contains(err.Error(), "notifications address is required") {
		t.Fatalf("Run error = %v, want notifications address validation", err)
	}
}

func TestNewAndServeLifecycle(t *testing.T) {
	previousDial := dialLenient
	dialLenient = func(context.Context, string, func(string, ...any), ...grpc.DialOption) *grpc.ClientConn {
		return nil
	}
	t.Cleanup(func() {
		dialLenient = previousDial
	})

	port := freeTCPPort(t)
	srv, err := New(context.Background(), RuntimeConfig{
		Port:              port,
		GameAddr:          "game:8082",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := srv.Addr(); got == "" {
		t.Fatal("expected listener address")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	waitForTCPReady(t, srv.Addr())
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop after cancellation")
	}
}

func TestServeRequiresContext(t *testing.T) {
	previousDial := dialLenient
	dialLenient = func(context.Context, string, func(string, ...any), ...grpc.DialOption) *grpc.ClientConn {
		return nil
	}
	t.Cleanup(func() {
		dialLenient = previousDial
	})

	srv, err := New(context.Background(), RuntimeConfig{
		Port:              freeTCPPort(t),
		GameAddr:          "game:8082",
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})

	if err := srv.Serve(nil); err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Serve error = %v, want context is required", err)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", listener.Addr())
	}
	return addr.Port
}

func waitForTCPReady(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not accept TCP connections at %s before timeout", addr)
}
