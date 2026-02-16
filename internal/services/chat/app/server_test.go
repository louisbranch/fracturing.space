package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewServerRequiresHTTPAddr(t *testing.T) {
	if _, err := NewServer(Config{}); err == nil {
		t.Fatal("expected error for empty HTTP address")
	}
}

func TestListenAndServeNilServer(t *testing.T) {
	var s *Server
	if err := s.ListenAndServe(context.Background()); err == nil {
		t.Fatal("expected error for nil server")
	}
}

func TestNewHandlerUpEndpoint(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/up", nil)

	NewHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}
	if strings.TrimSpace(rr.Body.String()) != "OK" {
		t.Fatalf("body = %q, want OK", rr.Body.String())
	}
}

func TestNewHandlerWSEndpoint(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ws", nil)

	NewHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestListenAndServeStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewServer(Config{HTTPAddr: "127.0.0.1:0"})
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
