package server

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewWithAddrContextRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(t.TempDir(), "social.db"))

	if _, err := NewWithAddrContext(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("NewWithAddrContext error = %v, want context is required", err)
	}
}

func TestRunWithAddrRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(t.TempDir(), "social.db"))

	if err := RunWithAddr(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("RunWithAddr error = %v, want context is required", err)
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(t.TempDir(), "social.db"))

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
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(t.TempDir(), "social.db"))

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}

	srv.Close()
	srv.Close()
}

func TestRunWithAddrStopsOnCancellation(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", filepath.Join(t.TempDir(), "social.db"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := RunWithAddr(ctx, "127.0.0.1:0"); err != nil && err != context.Canceled {
		t.Fatalf("RunWithAddr error = %v, want nil or context.Canceled", err)
	}
}
