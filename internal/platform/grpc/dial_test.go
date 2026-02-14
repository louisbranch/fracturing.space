package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDialWithHealthSuccess(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := DialWithHealth(ctx, nil, addr, time.Second, nil, DefaultClientDialOptions()...)
	if err != nil {
		t.Fatalf("dial with health: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close conn: %v", err)
	}
}

func TestDialWithHealthReturnsErrorWhenNotServing(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	conn, err := DialWithHealth(ctx, nil, addr, time.Second, nil, DefaultClientDialOptions()...)
	if err == nil {
		t.Fatal("expected error")
	}
	if conn != nil {
		_ = conn.Close()
		t.Fatal("expected nil connection on error")
	}
}

func TestDialWithHealthUsesDialTimeoutForHealth(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := DialWithHealth(ctx, nil, addr, 150*time.Millisecond, nil, DefaultClientDialOptions()...)
	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("expected dial timeout to bound health check, took %v", elapsed)
	}
}

func TestDialWithHealthErrorStages(t *testing.T) {
	t.Run("dial", func(t *testing.T) {
		dialer := DialerFunc(func(_ context.Context, _ string, _ ...gogrpc.DialOption) (*gogrpc.ClientConn, error) {
			return nil, fmt.Errorf("dial failure")
		})

		_, err := DialWithHealth(context.Background(), dialer, "unused", time.Second, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		var dialErr *DialError
		if !errors.As(err, &dialErr) {
			t.Fatalf("expected DialError, got %T", err)
		}
		if dialErr.Stage != DialStageConnect {
			t.Fatalf("expected stage %q, got %q", DialStageConnect, dialErr.Stage)
		}
	})

	t.Run("health", func(t *testing.T) {
		addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		defer stop()

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		_, err := DialWithHealth(ctx, nil, addr, time.Second, nil, DefaultClientDialOptions()...)
		if err == nil {
			t.Fatal("expected error")
		}
		var dialErr *DialError
		if !errors.As(err, &dialErr) {
			t.Fatalf("expected DialError, got %T", err)
		}
		if dialErr.Stage != DialStageHealth {
			t.Fatalf("expected stage %q, got %q", DialStageHealth, dialErr.Stage)
		}
	})
}

func TestDialErrorFormatting(t *testing.T) {
	wrapped := &DialError{Stage: DialStageConnect, Err: fmt.Errorf("boom")}
	if !strings.Contains(wrapped.Error(), "gRPC connect") {
		t.Fatalf("unexpected error: %s", wrapped.Error())
	}
	if wrapped.Unwrap() == nil {
		t.Fatal("expected wrapped error")
	}

	var nilErr *DialError
	if nilErr.Error() == "" {
		t.Fatal("expected fallback error message")
	}
	if nilErr.Unwrap() != nil {
		t.Fatal("expected nil unwrap for nil error")
	}
}

func TestDialerFuncCallsDelegate(t *testing.T) {
	called := false
	var gotAddr string
	var gotCtx context.Context

	dialer := DialerFunc(func(ctx context.Context, addr string, _ ...gogrpc.DialOption) (*gogrpc.ClientConn, error) {
		called = true
		gotAddr = addr
		gotCtx = ctx
		return nil, nil
	})

	if _, err := dialer.DialContext(context.Background(), "target"); err != nil {
		t.Fatalf("dial context: %v", err)
	}
	if !called {
		t.Fatal("expected dialer to be called")
	}
	if gotAddr != "target" {
		t.Fatalf("expected target addr, got %q", gotAddr)
	}
	if gotCtx == nil {
		t.Fatal("expected context to be passed")
	}
}
