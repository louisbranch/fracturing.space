package admin

import (
	"flag"
	"testing"
	"time"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != ":8082" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != "localhost:8080" {
		t.Fatalf("expected default grpc addr, got %q", cfg.GRPCAddr)
	}
	if cfg.AuthAddr != "localhost:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.AuthAddr)
	}
	if cfg.GRPCDialTimeout != 2*time.Second {
		t.Fatalf("expected default dial timeout, got %v", cfg.GRPCDialTimeout)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_ADMIN_ADDR", "env-admin")
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "env-game")
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "env-auth")

	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	args := []string{"-http-addr", "flag-admin", "-grpc-addr", "flag-game", "-auth-addr", "flag-auth"}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != "flag-admin" {
		t.Fatalf("expected flag http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != "flag-game" {
		t.Fatalf("expected flag grpc addr, got %q", cfg.GRPCAddr)
	}
	if cfg.AuthAddr != "flag-auth" {
		t.Fatalf("expected flag auth addr, got %q", cfg.AuthAddr)
	}
}
