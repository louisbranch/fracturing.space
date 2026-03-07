package seed

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	seedtool "github.com/louisbranch/fracturing.space/internal/tools/seed"
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

func TestParseConfigManifestModeDefaultsStatePath(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-manifest", "internal/tools/seed/manifests/local-dev.json"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.ManifestPath != "internal/tools/seed/manifests/local-dev.json" {
		t.Fatalf("manifest path = %q", cfg.ManifestPath)
	}
	if cfg.SeedStatePath == "" {
		t.Fatal("expected non-empty state path in manifest mode")
	}
	wantSuffix := filepath.Join(".tmp", "seed-state", "local-dev.state.json")
	if !strings.HasSuffix(cfg.SeedStatePath, wantSuffix) {
		t.Fatalf("state path = %q, want suffix %q", cfg.SeedStatePath, wantSuffix)
	}
}

func TestParseConfigManifestModeAcceptsNonLocalPathForRuntimeValidation(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-manifest", "internal/tools/seed/manifests/other.json"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.ManifestPath != "internal/tools/seed/manifests/other.json" {
		t.Fatalf("manifest path = %q", cfg.ManifestPath)
	}
}

func TestRun_ManifestModeUsesDeclarativeRunner(t *testing.T) {
	original := runDeclarativeManifestFn
	originalLookupHost := seedtool.LookupHost
	t.Cleanup(func() { runDeclarativeManifestFn = original })
	t.Cleanup(func() { seedtool.LookupHost = originalLookupHost })

	called := false
	runDeclarativeManifestFn = func(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
		called = true
		if cfg.ManifestPath != "internal/tools/seed/manifests/local-dev.json" {
			t.Fatalf("manifest path = %q", cfg.ManifestPath)
		}
		if cfg.SeedConfig.GRPCAddr != "127.0.0.1:8082" {
			t.Fatalf("expected normalized game addr, got %q", cfg.SeedConfig.GRPCAddr)
		}
		if cfg.SeedConfig.AuthAddr != "127.0.0.1:8083" {
			t.Fatalf("expected normalized auth addr, got %q", cfg.SeedConfig.AuthAddr)
		}
		if cfg.SocialAddr != "127.0.0.1:8090" {
			t.Fatalf("expected normalized social addr, got %q", cfg.SocialAddr)
		}
		if cfg.ListingAddr != "127.0.0.1:8091" {
			t.Fatalf("expected normalized listing addr, got %q", cfg.ListingAddr)
		}
		if cfg.SocialAddr == "" {
			t.Fatal("social addr should be set")
		}
		return nil
	}
	seedtool.LookupHost = func(_ context.Context, host string) ([]string, error) {
		if host == "" {
			return nil, fmt.Errorf("host required")
		}
		return nil, fmt.Errorf("dns disabled in test")
	}

	cfg := Config{
		Timeout:       10,
		ManifestPath:  "internal/tools/seed/manifests/local-dev.json",
		SeedStatePath: filepath.Join(".tmp", "seed-state", "local-dev.state.json"),
		SocialAddr:    "social:8090",
		ListingAddr:   "listing:8091",
		SeedConfig: seedtool.Config{
			GRPCAddr: "game:8082",
			AuthAddr: "auth:8083",
		},
	}

	if err := Run(context.Background(), cfg, io.Discard, io.Discard); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("expected manifest runner to be called")
	}
}

func TestRun_RejectsNonLocalManifestPath(t *testing.T) {
	cfg := Config{
		ManifestPath: "internal/tools/seed/manifests/prod.json",
	}
	if err := Run(context.Background(), cfg, io.Discard, io.Discard); err == nil {
		t.Fatal("expected error for non-local manifest path")
	}
}

func TestRun_RejectsMutationWithoutManifest(t *testing.T) {
	cfg := Config{
		ManifestPath: "",
	}
	if err := Run(context.Background(), cfg, io.Discard, io.Discard); err == nil {
		t.Fatal("expected error when running seed without a manifest")
	}
}

func TestParseConfigManifestModeRejectsGenerate(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	_, err := ParseConfig(fs, []string{
		"-manifest", "internal/tools/seed/manifests/local-dev.json",
		"-generate",
	})
	if err == nil {
		t.Fatal("expected parse error when -manifest and -generate are combined")
	}
}

func TestParseConfigDefaultSocialAddr(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.SocialAddr != "social:8090" {
		t.Fatalf("social addr = %q, want %q", cfg.SocialAddr, "social:8090")
	}
}
