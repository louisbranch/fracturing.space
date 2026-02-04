package web

import (
	"context"
	"testing"
	"time"
)

// TestNewServerRequiresHTTPAddr ensures a blank HTTP address fails fast.
func TestNewServerRequiresHTTPAddr(t *testing.T) {
	if _, err := NewServer(context.Background(), Config{}); err == nil {
		t.Fatal("expected error for empty HTTP address")
	}
}

// TestListenAndServeStopsOnCancel verifies the server exits on context cancel.
func TestListenAndServeStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewServer(ctx, Config{HTTPAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop on cancel")
	}
}
