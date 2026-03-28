package app

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	invitesqlite "github.com/louisbranch/fracturing.space/internal/services/invite/storage/sqlite"
	"google.golang.org/grpc"
)

func TestNewWithDepsRequiresRuntimeHooks(t *testing.T) {
	t.Parallel()

	testEnv := func() serverEnv { return serverEnv{DBPath: filepath.Join("testdata", "invite.db")} }
	tests := []struct {
		name string
		deps runtimeDeps
		want string
	}{
		{name: "env loader", deps: runtimeDeps{listen: net.Listen, openStore: openInviteStore, dialGame: noopInviteDialer, dialAuth: noopInviteDialer}, want: "invite server env loader is required"},
		{name: "listener", deps: runtimeDeps{loadEnv: testEnv, openStore: openInviteStore, dialGame: noopInviteDialer, dialAuth: noopInviteDialer}, want: "invite listener constructor is required"},
		{name: "store", deps: runtimeDeps{loadEnv: testEnv, listen: net.Listen, dialGame: noopInviteDialer, dialAuth: noopInviteDialer}, want: "invite store opener is required"},
		{name: "game dialer", deps: runtimeDeps{loadEnv: testEnv, listen: net.Listen, openStore: openInviteStore, dialAuth: noopInviteDialer}, want: "invite game dialer is required"},
		{name: "auth dialer", deps: runtimeDeps{loadEnv: testEnv, listen: net.Listen, openStore: openInviteStore, dialGame: noopInviteDialer}, want: "invite auth dialer is required"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newWithDeps(context.Background(), "127.0.0.1:0", "game:8082", "auth:8083", tc.deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newWithDeps error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNewWithDepsClosesListenerWhenStoreOpenFails(t *testing.T) {
	t.Parallel()

	listener := &inviteListenerStub{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 32001}}
	_, err := newWithDeps(context.Background(), "127.0.0.1:0", "game:8082", "auth:8083", runtimeDeps{
		loadEnv: func() serverEnv { return serverEnv{DBPath: filepath.Join("testdata", "invite.db")} },
		listen: func(string, string) (net.Listener, error) {
			return listener, nil
		},
		openStore: func(string) (*invitesqlite.Store, error) {
			return nil, errors.New("open boom")
		},
		dialGame: noopInviteDialer,
		dialAuth: noopInviteDialer,
		logf:     func(string, ...any) {},
	})
	if err == nil || !strings.Contains(err.Error(), "open boom") {
		t.Fatalf("newWithDeps error = %v, want store failure", err)
	}
	if !listener.closed {
		t.Fatal("listener closed = false, want true")
	}
}

func TestServeLifecycle(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_INVITE_DB_PATH", filepath.Join(t.TempDir(), "invite.db"))

	srv, err := NewWithAddr(context.Background(), "127.0.0.1:0", "127.0.0.1:1", "127.0.0.1:2")
	if err != nil {
		t.Fatalf("NewWithAddr: %v", err)
	}
	if got := srv.Addr(); got == "" {
		t.Fatal("expected listener address")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	waitForInviteTCPReady(t, srv.Addr())

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

func noopInviteDialer(context.Context, string, func(string, ...any)) (*grpc.ClientConn, error) {
	return nil, nil
}

type inviteListenerStub struct {
	addr   net.Addr
	closed bool
}

func (l *inviteListenerStub) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (l *inviteListenerStub) Close() error {
	l.closed = true
	return nil
}
func (l *inviteListenerStub) Addr() net.Addr { return l.addr }

func waitForInviteTCPReady(t *testing.T, addr string) {
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
