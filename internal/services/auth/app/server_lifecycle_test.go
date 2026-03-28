package server

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
)

func TestNewWithDepsRequiresRuntimeHooks(t *testing.T) {
	t.Parallel()

	testEnv := func() authServerEnv { return authServerEnv{DBPath: filepath.Join("testdata", "auth.db")} }
	tests := []struct {
		name string
		deps runtimeDeps
		want string
	}{
		{name: "env loader", deps: runtimeDeps{loadOAuthConfig: oauth.LoadConfigFromEnv, listen: net.Listen, openStore: openAuthStore, newOAuthServer: oauth.NewServer}, want: "auth server env loader is required"},
		{name: "oauth config", deps: runtimeDeps{loadEnv: testEnv, listen: net.Listen, openStore: openAuthStore, newOAuthServer: oauth.NewServer}, want: "auth oauth config loader is required"},
		{name: "listener", deps: runtimeDeps{loadEnv: testEnv, loadOAuthConfig: oauth.LoadConfigFromEnv, openStore: openAuthStore, newOAuthServer: oauth.NewServer}, want: "auth listener constructor is required"},
		{name: "store", deps: runtimeDeps{loadEnv: testEnv, loadOAuthConfig: oauth.LoadConfigFromEnv, listen: net.Listen, newOAuthServer: oauth.NewServer}, want: "auth store opener is required"},
		{name: "oauth server", deps: runtimeDeps{loadEnv: testEnv, loadOAuthConfig: oauth.LoadConfigFromEnv, listen: net.Listen, openStore: openAuthStore}, want: "auth oauth server constructor is required"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newWithDeps(0, "", tc.deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newWithDeps error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNewWithDepsClosesGRPCListenerWhenHTTPListenFails(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "auth.db")
	grpcListener := &authListenerStub{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 32031}}
	listenCalls := 0

	_, err := newWithDeps(0, "127.0.0.1:0", runtimeDeps{
		loadEnv:         func() authServerEnv { return authServerEnv{DBPath: storePath} },
		loadOAuthConfig: oauth.LoadConfigFromEnv,
		listen: func(network, address string) (net.Listener, error) {
			listenCalls++
			if listenCalls == 1 {
				return grpcListener, nil
			}
			return nil, errors.New("http listen boom")
		},
		openStore:      openAuthStore,
		newOAuthServer: oauth.NewServer,
		logf:           func(string, ...any) {},
	})
	if err == nil || !strings.Contains(err.Error(), "http listen boom") {
		t.Fatalf("newWithDeps error = %v, want HTTP listen failure", err)
	}
	if !grpcListener.closed {
		t.Fatal("grpc listener closed = false, want true")
	}
}

func TestServeLifecycleWithoutHTTP(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))

	srv, err := New(0, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := srv.Addr(); got == "" {
		t.Fatal("expected grpc listener address")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	waitForAuthTCPReady(t, loopbackDialAddr(t, srv.listener.Addr()))

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

func TestRunRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))

	if err := Run(nil, 0, ""); err == nil || err.Error() != "Context is required." {
		t.Fatalf("Run error = %v, want Context is required.", err)
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))

	srv, err := New(0, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})

	if err := srv.Serve(nil); err == nil || err.Error() != "Context is required." {
		t.Fatalf("Serve error = %v, want Context is required.", err)
	}
}

func TestServeLifecycleWithHTTP(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))

	srv, err := New(0, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if srv.httpListener == nil {
		t.Fatal("expected http listener")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	waitForAuthTCPReady(t, loopbackDialAddr(t, srv.listener.Addr()))
	waitForAuthTCPReady(t, loopbackDialAddr(t, srv.httpListener.Addr()))

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

func TestCloseIsIdempotent(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))

	srv, err := New(0, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.Close()
	srv.Close()
}

func waitForAuthTCPReady(t *testing.T, addr string) {
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

func loopbackDialAddr(t *testing.T, addr net.Addr) string {
	t.Helper()
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", addr)
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

type authListenerStub struct {
	addr   net.Addr
	closed bool
}

func (l *authListenerStub) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (l *authListenerStub) Close() error {
	l.closed = true
	return nil
}
func (l *authListenerStub) Addr() net.Addr { return l.addr }
