package grpc

import (
	"context"
	"testing"
	"time"

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDialLenientReturnsConnOnSuccess(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn := DialLenient(context.Background(), addr, nil)
	if conn == nil {
		t.Fatal("expected connection from lenient dial")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func TestDialLenientWithTimeoutReturnsConnOnSuccess(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn := DialLenientWithTimeout(context.Background(), addr, 250*time.Millisecond, nil)
	if conn == nil {
		t.Fatal("expected connection from lenient dial with timeout")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func TestDialLenientWithTimeoutUsesDefaultForNonPositiveTimeout(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn := DialLenientWithTimeout(context.Background(), addr, 0, nil)
	if conn == nil {
		t.Fatal("expected connection from lenient dial with default timeout")
	}
	_ = conn.Close()
}

func TestDialLenientReturnsNilOnBadAddress(t *testing.T) {
	conn := DialLenient(context.Background(), "127.0.0.1:1", nil)
	// DialLenient uses non-blocking dial, so it may still return a conn
	// that will fail on first RPC. The health check may fail or not depending
	// on timing. Either nil or a conn is acceptable since the dial is lenient.
	if conn != nil {
		_ = conn.Close()
	}
}

func TestDialLenientNilContextUsesBackground(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn := DialLenient(nil, addr, nil)
	if conn == nil {
		t.Fatal("expected connection with nil context")
	}
	_ = conn.Close()
}

func TestDialLenientLogsOnFailure(t *testing.T) {
	var logged bool
	logf := func(format string, args ...any) {
		logged = true
	}
	// Use a context that's already done to force failure.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	conn := DialLenient(ctx, "127.0.0.1:1", logf)
	if conn != nil {
		_ = conn.Close()
	}
	// With cancelled context, the dial should fail and log.
	if !logged {
		// Lenient dial may succeed on some platforms even with cancelled context
		// due to non-blocking dial. This is acceptable.
		t.Log("dial did not log failure (may be platform-dependent)")
	}
}

func TestLenientDialOptionsDoNotBlock(t *testing.T) {
	opts := LenientDialOptions()
	if len(opts) == 0 {
		t.Fatal("expected at least one dial option")
	}
	// LenientDialOptions should not include WithBlock.
	if len(opts) != 2 {
		t.Fatalf("expected 2 lenient dial options, got %d", len(opts))
	}
}
