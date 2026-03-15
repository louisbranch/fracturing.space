package mcp

import (
	"flag"
	"testing"

	mcpservice "github.com/louisbranch/fracturing.space/internal/services/mcp/service"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Addr != "game:8082" {
		t.Fatalf("expected default addr, got %q", cfg.Addr)
	}
	if cfg.HTTPAddr != "localhost:8085" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.RegistrationProfile != "" {
		t.Fatalf("expected empty default registration profile, got %q", cfg.RegistrationProfile)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "env-game")
	t.Setenv("FRACTURING_SPACE_MCP_HTTP_ADDR", "env-http")

	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	args := []string{"-addr", "flag-game", "-http-addr", "flag-http"}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Addr != "flag-game" {
		t.Fatalf("expected flag addr, got %q", cfg.Addr)
	}
	if cfg.HTTPAddr != "flag-http" {
		t.Fatalf("expected flag http addr, got %q", cfg.HTTPAddr)
	}
}

func TestParseConfigReadsHarnessProfileFromEnv(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_MCP_PROFILE", string(mcpservice.RegistrationProfileHarness))

	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.RegistrationProfile != mcpservice.RegistrationProfileHarness {
		t.Fatalf("expected harness profile, got %q", cfg.RegistrationProfile)
	}
}
