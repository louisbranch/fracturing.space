package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadServerEnvAndWrappers(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", "")
		cfg := loadServerEnv()
		if cfg.DBPath != filepath.Join("data", "status.db") {
			t.Fatalf("loadServerEnv() default path = %q", cfg.DBPath)
		}
	})

	t.Run("configured path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "status.db")
		t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", path)
		cfg := loadServerEnv()
		if cfg.DBPath != path {
			t.Fatalf("loadServerEnv() configured path = %q", cfg.DBPath)
		}
	})
}

func TestNewRunAddrAndServeBranches(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_STATUS_DB_PATH", filepath.Join(t.TempDir(), "status.db"))

	srv, err := New(0)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	if got := srv.Addr(); got == "" {
		t.Fatal("New().Addr() = empty, want listener address")
	}

	var nilServer *Server
	if err := nilServer.Serve(context.Background()); err == nil || err.Error() != "server is nil" {
		t.Fatalf("(*Server)(nil).Serve() error = %v, want server is nil", err)
	}
	if got := nilServer.Addr(); got != "" {
		t.Fatalf("(*Server)(nil).Addr() = %q, want empty", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, 0)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after cancellation")
	}
}

func TestOpenStoreBranches(t *testing.T) {
	t.Run("creates nested directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "status.db")
		store, err := openStore(path)
		if err != nil {
			t.Fatalf("openStore() error = %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			t.Fatalf("storage dir stat error = %v", err)
		}
	})

	t.Run("directory creation failure", func(t *testing.T) {
		parentFile := filepath.Join(t.TempDir(), "file-parent")
		if err := os.WriteFile(parentFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		_, err := openStore(filepath.Join(parentFile, "status.db"))
		if err == nil || !strings.Contains(err.Error(), "create storage dir") {
			t.Fatalf("openStore() error = %v, want create-dir failure", err)
		}
	})
}
