package app

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestNormalizeGRPCServeError(t *testing.T) {
	if err := normalizeGRPCServeError(nil); err != nil {
		t.Fatalf("normalizeGRPCServeError(nil) = %v, want nil", err)
	}
	if err := normalizeGRPCServeError(grpc.ErrServerStopped); err != nil {
		t.Fatalf("normalizeGRPCServeError(ErrServerStopped) = %v, want nil", err)
	}
	cause := errors.New("boom")
	err := normalizeGRPCServeError(cause)
	if !errors.Is(err, cause) {
		t.Fatalf("normalizeGRPCServeError() error = %v, want wrapped cause %v", err, cause)
	}
	if !strings.Contains(err.Error(), "serve gRPC") {
		t.Fatalf("normalizeGRPCServeError() error = %v, want serve gRPC context", err)
	}
}

func TestRunGRPCServeLoop_ReturnsServeError(t *testing.T) {
	cause := errors.New("serve failed")
	shutdownCalls := atomic.Int32{}
	err := runGRPCServeLoop(
		context.Background(),
		func() error { return cause },
		func() { shutdownCalls.Add(1) },
	)
	if !errors.Is(err, cause) {
		t.Fatalf("runGRPCServeLoop() error = %v, want %v", err, cause)
	}
	if shutdownCalls.Load() != 0 {
		t.Fatalf("shutdown calls = %d, want 0 when serve exits first", shutdownCalls.Load())
	}
}

func TestRunGRPCServeLoop_ContextCancelTriggersShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCalls := atomic.Int32{}
	release := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- runGRPCServeLoop(
			ctx,
			func() error {
				<-release
				return grpc.ErrServerStopped
			},
			func() {
				shutdownCalls.Add(1)
				close(release)
			},
		)
	}()

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runGRPCServeLoop() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected serve loop to exit after context cancellation")
	}

	if shutdownCalls.Load() != 1 {
		t.Fatalf("shutdown calls = %d, want 1", shutdownCalls.Load())
	}
}

func TestRunGRPCServeLoop_RequiresServeFunction(t *testing.T) {
	err := runGRPCServeLoop(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error when serve function is nil")
	}
	if !strings.Contains(err.Error(), "serve function") {
		t.Fatalf("runGRPCServeLoop() error = %v, want serve function context", err)
	}
}
