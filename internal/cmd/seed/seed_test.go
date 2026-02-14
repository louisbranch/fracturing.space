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
	cfg, err := ParseConfig(fs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Preset != generator.PresetDemo {
		t.Fatalf("expected demo preset, got %q", cfg.Preset)
	}
	if cfg.SeedConfig.RepoRoot == "" {
		t.Fatal("expected repo root to be set")
	}
	if _, err := os.Stat(filepath.Join(cfg.SeedConfig.RepoRoot, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in repo root: %v", err)
	}
}

func TestParseConfigListFlag(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-list"}, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.List {
		t.Fatal("expected list flag to be true")
	}
}
