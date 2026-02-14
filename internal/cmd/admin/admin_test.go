package admin

import (
	"flag"
	"testing"
	"time"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != defaultGRPCAddr {
		t.Fatalf("expected default grpc addr, got %q", cfg.GRPCAddr)
	}
	if cfg.AuthAddr != defaultAuthAddr {
		t.Fatalf("expected default auth addr, got %q", cfg.AuthAddr)
	}
	if cfg.GRPCDialTimeout != 2*time.Second {
		t.Fatalf("expected default dial timeout, got %v", cfg.GRPCDialTimeout)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	lookup := func(key string) (string, bool) {
		switch key {
		case "FRACTURING_SPACE_ADMIN_ADDR":
			return "env-admin", true
		case "FRACTURING_SPACE_GAME_ADDR":
			return "env-game", true
		case "FRACTURING_SPACE_AUTH_ADDR":
			return "env-auth", true
		default:
			return "", false
		}
	}
	args := []string{"-http-addr", "flag-admin", "-grpc-addr", "flag-game", "-auth-addr", "flag-auth"}
	cfg, err := ParseConfig(fs, args, lookup)
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
