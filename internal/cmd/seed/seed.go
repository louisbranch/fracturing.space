package seed

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/tools/seed"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/generator"
)

// Config holds seed command configuration.
type Config struct {
	SeedConfig seed.Config
	List       bool
	Generate   bool
	Preset     generator.Preset
	Seed       int64
	Campaigns  int
}

// EnvLookup returns the value for a key when present.
type EnvLookup func(string) (string, bool)

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string, lookup EnvLookup) (Config, error) {
	seedCfg := seed.DefaultConfig()
	seedCfg.AuthAddr = envOrDefault(lookup, []string{"FRACTURING_SPACE_AUTH_ADDR"}, seedCfg.AuthAddr)
	var list bool
	var generate bool
	var preset string
	var seedVal int64
	var campaigns int

	fs.StringVar(&seedCfg.GRPCAddr, "grpc-addr", seedCfg.GRPCAddr, "game server address")
	fs.StringVar(&seedCfg.AuthAddr, "auth-addr", seedCfg.AuthAddr, "auth server address")
	fs.StringVar(&seedCfg.Scenario, "scenario", "", "run specific scenario (default: all)")
	fs.BoolVar(&seedCfg.Verbose, "v", false, "verbose output")
	fs.BoolVar(&list, "list", false, "list available scenarios")
	fs.BoolVar(&generate, "generate", false, "use dynamic generation instead of fixtures")
	fs.StringVar(&preset, "preset", string(generator.PresetDemo), "generation preset (demo, variety, session-heavy, stress-test)")
	fs.Int64Var(&seedVal, "seed", 0, "random seed for reproducibility (0 = random)")
	fs.IntVar(&campaigns, "campaigns", 0, "number of campaigns to generate (0 = use preset default)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	root, err := repoRoot()
	if err != nil {
		return Config{}, err
	}
	seedCfg.RepoRoot = root

	return Config{
		SeedConfig: seedCfg,
		List:       list,
		Generate:   generate,
		Preset:     generator.Preset(preset),
		Seed:       seedVal,
		Campaigns:  campaigns,
	}, nil
}

// Run executes the seed command.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	if cfg.List {
		scenarios, err := seed.ListScenarios(cfg.SeedConfig)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "Available scenarios:")
		for _, name := range scenarios {
			fmt.Fprintf(out, "  %s\n", name)
		}
		fmt.Fprintln(out, "\nAvailable presets (for -generate):")
		fmt.Fprintln(out, "  demo         - Rich single campaign with full party")
		fmt.Fprintln(out, "  variety      - 8 campaigns across all statuses/modes")
		fmt.Fprintln(out, "  session-heavy - Few campaigns with many sessions")
		fmt.Fprintln(out, "  stress-test  - 50 minimal campaigns")
		return nil
	}

	if cfg.Generate {
		if err := validatePreset(cfg.Preset); err != nil {
			return err
		}
		genCfg := generator.Config{
			GRPCAddr:  cfg.SeedConfig.GRPCAddr,
			AuthAddr:  cfg.SeedConfig.AuthAddr,
			Preset:    cfg.Preset,
			Seed:      cfg.Seed,
			Campaigns: cfg.Campaigns,
			Verbose:   cfg.SeedConfig.Verbose,
		}
		gen, err := generator.New(ctx, genCfg)
		if err != nil {
			return err
		}
		defer gen.Close()

		if err := gen.Run(ctx); err != nil {
			return err
		}
		return nil
	}

	if err := seed.Run(ctx, cfg.SeedConfig); err != nil {
		return err
	}
	return nil
}

func validatePreset(preset generator.Preset) error {
	validPresets := []generator.Preset{
		generator.PresetDemo,
		generator.PresetVariety,
		generator.PresetSessionHeavy,
		generator.PresetStressTest,
	}
	for _, p := range validPresets {
		if preset == p {
			return nil
		}
	}
	return fmt.Errorf("unknown preset %q (valid presets: demo, variety, session-heavy, stress-test)", preset)
}

func repoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve runtime caller")
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found from %s", filename)
}

func envOrDefault(lookup EnvLookup, keys []string, fallback string) string {
	for _, key := range keys {
		if lookup == nil {
			break
		}
		value, ok := lookup(key)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}
