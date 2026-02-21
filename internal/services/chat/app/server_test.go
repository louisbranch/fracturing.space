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

func TestNewServerWithContextRequiresContext(t *testing.T) {
	if _, err := NewServerWithContext(nil, Config{HTTPAddr: "127.0.0.1:0"}); err == nil {
		t.Fatal("expected error for nil context")
	}
}

func TestDialGameGRPCNilContextReturnsError(t *testing.T) {
	_, err := dialGameGRPC(nil, Config{
		GameAddr: "127.0.0.1:1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("unexpected error: %v", err)
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

func TestListenAndServeRequiresContext(t *testing.T) {
	server, err := NewServer(Config{HTTPAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.ListenAndServe(nil); err == nil {
		t.Fatal("expected error")
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

func TestMustJSONReturnsNilWithoutPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("mustJSON panicked: %v", r)
		}
	}()

	if got := mustJSON(make(chan int)); got != nil {
		t.Fatalf("mustJSON = %v, want nil", got)
	}
}

func TestServerCloseStopsCampaignUpdateSubscriptionWorker(t *testing.T) {
	done := make(chan struct{})
	stopped := false
	server := &Server{
		campaignUpdateSubscriptionDone: done,
		campaignUpdateSubscriptionStop: func() {
			stopped = true
			close(done)
		},
	}

	server.Close()

	if !stopped {
		t.Fatalf("expected campaign update subscription stop to be called")
	}
}
