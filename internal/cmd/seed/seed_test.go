package seed

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/tools/seed/generator"
)

func TestValidatePreset(t *testing.T) {
	if err := validatePreset(generator.PresetDemo); err != nil {
		t.Fatalf("expected demo to be valid: %v", err)
	}
	if err := validatePreset("unknown"); err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Preset != generator.PresetDemo {
		t.Fatalf("expected demo preset, got %q", cfg.Preset)
	}
	if cfg.SeedConfig.AuthAddr != "auth:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.SeedConfig.AuthAddr)
	}
	if cfg.SeedConfig.GRPCAddr != "game:8082" {
		t.Fatalf("expected default game grpc addr, got %q", cfg.SeedConfig.GRPCAddr)
	}
	if cfg.SeedConfig.RepoRoot == "" {
		t.Fatal("expected repo root to be set")
	}
	if _, err := os.Stat(filepath.Join(cfg.SeedConfig.RepoRoot, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in repo root: %v", err)
	}
}

func TestParseConfigReadsServiceAddrEnvOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "localhost:18082")
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "localhost:18083")

	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.SeedConfig.GRPCAddr != "localhost:18082" {
		t.Fatalf("expected env game grpc addr, got %q", cfg.SeedConfig.GRPCAddr)
	}
	if cfg.SeedConfig.AuthAddr != "localhost:18083" {
		t.Fatalf("expected env auth grpc addr, got %q", cfg.SeedConfig.AuthAddr)
	}
}

func TestParseConfigListFlag(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-list"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.List {
		t.Fatal("expected list flag to be true")
	}
}
