package grpc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/status"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	testRetryDelay          = 10 * time.Millisecond
	testMaxRetryDelay       = 20 * time.Millisecond
	testHealthyPollInterval = 20 * time.Millisecond
)

func silentDialOpts() []gogrpc.DialOption {
	return []gogrpc.DialOption{
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
	}
}

func nopLogf(string, ...any) {}

func TestManagedConn_optional_returns_immediately(t *testing.T) {
	t.Parallel()

	healthy := make(chan struct{})
	mc, err := newTestManagedConn(t, ModeOptional, func(ctx context.Context, _ *gogrpc.ClientConn) error {
		select {
		case <-healthy:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err != nil {
		t.Fatalf("NewManagedConn: %v", err)
	}
	defer mc.Close()

	if mc.Ready() {
		t.Fatal("expected not ready before health check passes")
	}
	if mc.Conn() == nil {
		t.Fatal("conn must be non-nil")
	}

	// Signal healthy.
	close(healthy)
	if err := mc.WaitReady(ctxWithTimeout(t, 2*time.Second)); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if !mc.Ready() {
		t.Fatal("expected ready after health check passes")
	}
}

func TestManagedConn_required_blocks_until_healthy(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	mc, err := newTestManagedConn(t, ModeRequired, func(_ context.Context, _ *gogrpc.ClientConn) error {
		if calls.Add(1) < 2 {
			return errors.New("not ready")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("NewManagedConn: %v", err)
	}
	defer mc.Close()

	if !mc.Ready() {
		t.Fatal("expected ready after ModeRequired construction")
	}
}

func TestManagedConn_required_context_cancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := NewManagedConn(ctx, ManagedConnConfig{
		Name:     "test",
		Addr:     "127.0.0.1:1",
		Mode:     ModeRequired,
		DialOpts: silentDialOpts(),
		Logf:     nopLogf,
		checkHealth: func(ctx context.Context, _ *gogrpc.ClientConn) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Hour):
				return errors.New("timeout")
			}
		},
	})
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
}

func TestManagedConn_health_transitions_report_status(t *testing.T) {
	t.Parallel()

	reporter := status.NewReporter("test", nil, status.WithPushInterval(time.Hour))
	reporter.Register("test.dep", status.Unavailable)

	healthy := atomic.Bool{}
	mc, err := newTestManagedConnWithReporter(t, ModeOptional, reporter, "test.dep", func(_ context.Context, _ *gogrpc.ClientConn) error {
		if healthy.Load() {
			return nil
		}
		return errors.New("down")
	})
	if err != nil {
		t.Fatalf("NewManagedConn: %v", err)
	}
	defer mc.Close()

	waitForStatusCapability(t, reporter, "test.dep", status.Unavailable, 200*time.Millisecond)

	healthy.Store(true)
	if err := mc.WaitReady(ctxWithTimeout(t, 5*time.Second)); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	waitForStatusCapability(t, reporter, "test.dep", status.Operational, 200*time.Millisecond)

	// Simulate degradation.
	healthy.Store(false)
	waitForStatusCapability(t, reporter, "test.dep", status.Unavailable, 200*time.Millisecond)
}

func TestManagedConn_close_stops_monitor(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	mc, err := newTestManagedConn(t, ModeOptional, func(_ context.Context, _ *gogrpc.ClientConn) error {
		calls.Add(1)
		return errors.New("down")
	})
	if err != nil {
		t.Fatalf("NewManagedConn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if err := mc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	countAtClose := calls.Load()
	time.Sleep(100 * time.Millisecond)

	if calls.Load() != countAtClose {
		t.Fatal("monitor continued after Close")
	}
}

func TestManagedConn_requires_address(t *testing.T) {
	t.Parallel()

	_, err := NewManagedConn(context.Background(), ManagedConnConfig{
		Name:     "test",
		DialOpts: silentDialOpts(),
		Logf:     nopLogf,
	})
	if err == nil {
		t.Fatal("expected error for empty address")
	}
}

// --- helpers ---

func newTestManagedConn(
	t *testing.T,
	mode ManagedConnMode,
	check func(context.Context, *gogrpc.ClientConn) error,
) (*ManagedConn, error) {
	t.Helper()
	return newTestManagedConnWithReporter(t, mode, nil, "", check)
}

func newTestManagedConnWithReporter(
	t *testing.T,
	mode ManagedConnMode,
	reporter *status.Reporter,
	capability string,
	check func(context.Context, *gogrpc.ClientConn) error,
) (*ManagedConn, error) {
	t.Helper()

	ctx := ctxWithTimeout(t, 10*time.Second)
	return NewManagedConn(ctx, ManagedConnConfig{
		Name:                "test",
		Addr:                "127.0.0.1:1",
		Mode:                mode,
		DialOpts:            silentDialOpts(),
		Logf:                nopLogf,
		StatusReporter:      reporter,
		StatusCapability:    capability,
		retryDelay:          testRetryDelay,
		maxRetryDelay:       testMaxRetryDelay,
		healthyPollInterval: testHealthyPollInterval,
		checkHealth:         check,
	})
}

func ctxWithTimeout(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func waitForStatusCapability(t *testing.T, reporter *status.Reporter, capability string, want status.CapabilityStatus, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, capabilityState := range reporter.Snapshot() {
			if capabilityState.Name == capability && capabilityState.Status == want {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}

	for _, capabilityState := range reporter.Snapshot() {
		if capabilityState.Name == capability {
			t.Fatalf("capability %s status = %v, want %v", capability, capabilityState.Status, want)
		}
	}
	t.Fatalf("capability %s not found in reporter snapshot", capability)
}
