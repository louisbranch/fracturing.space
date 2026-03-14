package server

import (
	"context"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

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
