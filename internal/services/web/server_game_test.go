package web

import (
	"testing"
	"time"
)

func TestNewServerDoesNotFailWhenGameDialFails(t *testing.T) {
	srv, err := NewServer(Config{
		HTTPAddr:        "127.0.0.1:0",
		AuthBaseURL:     "http://auth.local",
		OAuthClientID:   "fracturing-space",
		CallbackURL:     "http://localhost:8080/auth/callback",
		AuthAddr:        "",
		GameAddr:        "127.0.0.1:1",
		GRPCDialTimeout: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if srv == nil {
		t.Fatal("expected server")
	}
	srv.Close()
}
