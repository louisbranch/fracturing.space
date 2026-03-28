package app

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	invitesqlite "github.com/louisbranch/fracturing.space/internal/services/invite/storage/sqlite"
	"google.golang.org/grpc"
)

func TestLoadServerEnvBranches(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_INVITE_DB_PATH", "")
		cfg := loadServerEnv()
		if cfg.DBPath != filepath.Join("data", "invite.db") {
			t.Fatalf("loadServerEnv() default path = %q", cfg.DBPath)
		}
	})

	t.Run("configured path", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_INVITE_DB_PATH", filepath.Join("tmp", "invite.db"))
		cfg := loadServerEnv()
		if cfg.DBPath != filepath.Join("tmp", "invite.db") {
			t.Fatalf("loadServerEnv() configured path = %q", cfg.DBPath)
		}
	})
}

func TestRunAndAddrBranches(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_INVITE_DB_PATH", filepath.Join(t.TempDir(), "invite.db"))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, Config{Port: 0, GameAddr: "127.0.0.1:1", AuthAddr: "127.0.0.1:2"})
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

	var nilServer *Server
	if got := nilServer.Addr(); got != "" {
		t.Fatalf("(*Server)(nil).Addr() = %q, want empty", got)
	}
}

func TestNewWithDepsDialFailureClosesListener(t *testing.T) {
	baseDir := t.TempDir()
	newStore := func(t *testing.T) func(string) (*invitesqlite.Store, error) {
		return func(_ string) (*invitesqlite.Store, error) {
			name := strings.ReplaceAll(t.Name(), "/", "_")
			return invitesqlite.Open(filepath.Join(baseDir, name+".db"))
		}
	}

	t.Run("game dial failure", func(t *testing.T) {
		listener := &inviteListenerStub{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 32011}}
		_, err := newWithDeps(context.Background(), "127.0.0.1:0", "game:8082", "auth:8083", runtimeDeps{
			loadEnv: func() serverEnv { return serverEnv{DBPath: filepath.Join(baseDir, "unused.db")} },
			listen: func(string, string) (net.Listener, error) {
				return listener, nil
			},
			openStore: newStore(t),
			dialGame: func(context.Context, string, func(string, ...any)) (*grpc.ClientConn, error) {
				return nil, errors.New("game dial boom")
			},
			dialAuth: noopInviteDialer,
			logf:     func(string, ...any) {},
		})
		if err == nil || !strings.Contains(err.Error(), "dial game service") {
			t.Fatalf("newWithDeps(game dial) error = %v", err)
		}
		if !listener.closed {
			t.Fatal("listener closed = false, want true")
		}
	})

	t.Run("auth dial failure", func(t *testing.T) {
		listener := &inviteListenerStub{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 32012}}
		_, err := newWithDeps(context.Background(), "127.0.0.1:0", "game:8082", "auth:8083", runtimeDeps{
			loadEnv: func() serverEnv { return serverEnv{DBPath: filepath.Join(baseDir, "unused.db")} },
			listen: func(string, string) (net.Listener, error) {
				return listener, nil
			},
			openStore: newStore(t),
			dialGame:  noopInviteDialer,
			dialAuth: func(context.Context, string, func(string, ...any)) (*grpc.ClientConn, error) {
				return nil, errors.New("auth dial boom")
			},
			logf: func(string, ...any) {},
		})
		if err == nil || !strings.Contains(err.Error(), "dial auth service") {
			t.Fatalf("newWithDeps(auth dial) error = %v", err)
		}
		if !listener.closed {
			t.Fatal("listener closed = false, want true")
		}
	})
}

func TestOpenInviteStoreBranches(t *testing.T) {
	t.Run("creates nested directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "invite.db")
		store, err := openInviteStore(path)
		if err != nil {
			t.Fatalf("openInviteStore() error = %v", err)
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
		_, err := openInviteStore(filepath.Join(parentFile, "invite.db"))
		if err == nil || !strings.Contains(err.Error(), "create storage dir") {
			t.Fatalf("openInviteStore() error = %v, want create-dir failure", err)
		}
	})
}
