package server

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWithAddrInitializesServer(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}
	if srv == nil {
		t.Fatal("expected server instance")
	}
	if addr := srv.Addr(); addr == "" {
		t.Fatal("expected non-empty listener address")
	}
	srv.Close()
}

func TestNewWithAddrContextRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	if _, err := NewWithAddrContext(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("NewWithAddrContext error = %v, want context is required", err)
	}
}

func TestRunWithAddrRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	if err := RunWithAddr(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("RunWithAddr error = %v, want context is required", err)
	}
}

func TestNewWithAddrContextHonorsCancellation(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewWithAddrContext(ctx, "127.0.0.1:0")
	if err == nil {
		t.Fatal("NewWithAddrContext error = nil, want cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("NewWithAddrContext error = %v, want context canceled", err)
	}
}

func TestServeStopsOnContextCancellation(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop after context cancellation")
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})

	if err := srv.Serve(nil); err == nil || err.Error() != "context is required" {
		t.Fatalf("Serve error = %v, want context is required", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}

	srv.Close()
	srv.Close()
}
