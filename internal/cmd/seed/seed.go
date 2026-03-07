// Package seed parses seed command flags and executes fixture / generation workflows.
package seed

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/tools/seed"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/declarative"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/generator"
)

const localManifestPath = "internal/tools/seed/manifests/local-dev.json"

// Config holds seed command configuration.
type Config struct {
	SeedConfig    seed.Config
	Timeout       time.Duration
	List          bool
	Generate      bool
	DiscoveryAddr string
	SocialAddr    string
	ManifestPath  string
	SeedStatePath string
	Preset        generator.Preset
	Seed          int64
	Campaigns     int
}

// seedEnv holds env-tagged fields for the seed command.
type seedEnv struct {
	GameAddr      string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AuthAddr      string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	DiscoveryAddr string        `env:"FRACTURING_SPACE_DISCOVERY_ADDR"`
	SocialAddr    string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	Timeout       time.Duration `env:"FRACTURING_SPACE_SEED_TIMEOUT" envDefault:"10m"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var se seedEnv
	if err := entrypoint.ParseConfig(&se); err != nil {
		return Config{}, err
	}

	seedCfg := seed.DefaultConfig()
	seedCfg.GRPCAddr = serviceaddr.OrDefaultGRPCAddr(se.GameAddr, serviceaddr.ServiceGame)
	seedCfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(se.AuthAddr, serviceaddr.ServiceAuth)
	discoveryAddr := serviceaddr.OrDefaultGRPCAddr(se.DiscoveryAddr, serviceaddr.ServiceDiscovery)
	socialAddr := serviceaddr.OrDefaultGRPCAddr(se.SocialAddr, serviceaddr.ServiceSocial)
	timeout := se.Timeout
	var list bool
	var generate bool
	var preset string
	var seedVal int64
	var campaigns int
	var manifestPath string
	var seedStatePath string

	fs.StringVar(&seedCfg.GRPCAddr, "grpc-addr", seedCfg.GRPCAddr, "game server address")
	fs.StringVar(&seedCfg.AuthAddr, "auth-addr", seedCfg.AuthAddr, "auth server address")
	fs.StringVar(&discoveryAddr, "discovery-addr", discoveryAddr, "discovery server address")
	fs.StringVar(&socialAddr, "social-addr", socialAddr, "social server address")
	fs.DurationVar(&timeout, "timeout", timeout, "overall timeout")
	fs.StringVar(&seedCfg.Scenario, "scenario", "", "run specific scenario (default: all)")
	fs.BoolVar(&seedCfg.Verbose, "v", false, "verbose output")
	fs.BoolVar(&list, "list", false, "list available scenarios")
	fs.BoolVar(&generate, "generate", false, "use dynamic generation instead of fixtures")
	fs.StringVar(&preset, "preset", string(generator.PresetDemo), "generation preset (demo, variety, session-heavy, stress-test)")
	fs.Int64Var(&seedVal, "seed", 0, "random seed for reproducibility (0 = random)")
	fs.IntVar(&campaigns, "campaigns", 0, "number of campaigns to generate (0 = use preset default)")
	fs.StringVar(&manifestPath, "manifest", "", "run declarative seed manifest from JSON file")
	fs.StringVar(&seedStatePath, "seed-state", "", "path to declarative seed state file")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}

	root, err := repoRoot()
	if err != nil {
		return Config{}, err
	}
	seedCfg.RepoRoot = root
	manifestPath = strings.TrimSpace(manifestPath)
	seedStatePath = strings.TrimSpace(seedStatePath)
	if manifestPath != "" && seedStatePath == "" {
		seedStatePath = defaultSeedStatePathForManifest(manifestPath)
	}
	if manifestPath != "" && generate {
		return Config{}, fmt.Errorf("cannot use -manifest with -generate")
	}
	if manifestPath != "" && seedCfg.Scenario != "" {
		return Config{}, fmt.Errorf("cannot use -manifest with -scenario")
	}

	return Config{
		SeedConfig:    seedCfg,
		Timeout:       timeout,
		List:          list,
		Generate:      generate,
		DiscoveryAddr: discoveryAddr,
		SocialAddr:    socialAddr,
		ManifestPath:  manifestPath,
		SeedStatePath: seedStatePath,
		Preset:        generator.Preset(preset),
		Seed:          seedVal,
		Campaigns:     campaigns,
	}, nil
}

func validateSeedMode(cfg Config) error {
	if cfg.List {
		return nil
	}
	if strings.TrimSpace(cfg.ManifestPath) == "" {
		return fmt.Errorf("seed command is restricted to local manifest mode; use -manifest=%q", localManifestPath)
	}
	if strings.TrimSpace(cfg.ManifestPath) != localManifestPath {
		return fmt.Errorf("seed manifest path is restricted to %q in this build", localManifestPath)
	}
	return nil
}

// Run executes the seed command across dynamic generation or fixture replay.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if err := validateSeedMode(cfg); err != nil {
		return err
	}
	runCtx := ctx
	var cancel func()
	if cfg.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, cfg.Timeout)
		defer cancel()
	}

	return entrypoint.RunWithTelemetry(runCtx, entrypoint.ServiceSeed, func(runCtx context.Context) error {
		cfg = normalizeSeedAddrs(cfg)
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
			fmt.Fprintln(out, "\nDeclarative manifest mode:")
			fmt.Fprintln(out, "  -manifest=internal/tools/seed/manifests/local-dev.json")
			fmt.Fprintln(out, "  -seed-state=.tmp/seed-state/local-dev.state.json")
			return nil
		}
		if strings.TrimSpace(cfg.ManifestPath) != "" {
			return runDeclarativeManifestFn(runCtx, cfg, out, errOut)
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
			gen, err := generator.New(runCtx, genCfg)
			if err != nil {
				return err
			}
			defer gen.Close()

			return gen.Run(runCtx)
		}
		return seed.Run(runCtx, cfg.SeedConfig)
	})
}

var runDeclarativeManifestFn = runDeclarativeManifest

func runDeclarativeManifest(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if strings.TrimSpace(cfg.ManifestPath) == "" {
		return fmt.Errorf("manifest path is required")
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	gameAddr := cfg.SeedConfig.GRPCAddr
	authAddr := cfg.SeedConfig.AuthAddr
	socialAddr := cfg.SocialAddr
	discoveryAddr := cfg.DiscoveryAddr
	if cfg.SeedConfig.Verbose {
		fmt.Fprintf(errOut, "seed: resolved gRPC endpoints\n")
		fmt.Fprintf(errOut, "seed: game %q -> %q\n", cfg.SeedConfig.GRPCAddr, gameAddr)
		fmt.Fprintf(errOut, "seed: auth %q -> %q\n", cfg.SeedConfig.AuthAddr, authAddr)
		fmt.Fprintf(errOut, "seed: social %q -> %q\n", cfg.SocialAddr, socialAddr)
		fmt.Fprintf(errOut, "seed: discovery %q -> %q\n", cfg.DiscoveryAddr, discoveryAddr)
	}
	if err := waitForTCP(ctx, "game", gameAddr, errOut, cfg.SeedConfig.Verbose); err != nil {
		return err
	}
	if err := waitForTCP(ctx, "auth", authAddr, errOut, cfg.SeedConfig.Verbose); err != nil {
		return err
	}
	if err := waitForTCP(ctx, "social", socialAddr, errOut, cfg.SeedConfig.Verbose); err != nil {
		return err
	}
	if err := waitForTCP(ctx, "discovery", discoveryAddr, errOut, cfg.SeedConfig.Verbose); err != nil {
		return err
	}
	runner, err := declarative.NewGRPCRunner(
		declarative.Config{
			ManifestPath: cfg.ManifestPath,
			StatePath:    cfg.SeedStatePath,
			Verbose:      cfg.SeedConfig.Verbose,
		},
		declarative.DialConfig{
			GameAddr:      gameAddr,
			AuthAddr:      authAddr,
			SocialAddr:    socialAddr,
			DiscoveryAddr: discoveryAddr,
		},
	)
	if err != nil {
		return err
	}
	defer func() { _ = runner.Close() }()

	if err := runner.Run(ctx); err != nil {
		return err
	}
	if out != nil {
		fmt.Fprintf(out, "Applied declarative seed manifest: %s (state: %s)\n", cfg.ManifestPath, cfg.SeedStatePath)
	}
	return nil
}

func defaultSeedStatePathForManifest(manifestPath string) string {
	base := strings.TrimSpace(filepath.Base(manifestPath))
	if base == "" {
		return filepath.Join(".tmp", "seed-state", "manifest.state.json")
	}
	ext := filepath.Ext(base)
	name := strings.TrimSpace(strings.TrimSuffix(base, ext))
	if name == "" {
		name = "manifest"
	}
	return filepath.Join(".tmp", "seed-state", name+".state.json")
}

func normalizeSeedAddrs(cfg Config) Config {
	cfg.SeedConfig.GRPCAddr = seed.ResolveLocalFallbackAddr(cfg.SeedConfig.GRPCAddr)
	cfg.SeedConfig.AuthAddr = seed.ResolveLocalFallbackAddr(cfg.SeedConfig.AuthAddr)
	cfg.SocialAddr = seed.ResolveLocalFallbackAddr(cfg.SocialAddr)
	cfg.DiscoveryAddr = seed.ResolveLocalFallbackAddr(cfg.DiscoveryAddr)
	return cfg
}

func waitForTCP(ctx context.Context, label, addr string, out io.Writer, verbose bool) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("%s address is required", label)
	}
	const attemptTimeout = 500 * time.Millisecond
	attempt := 0
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("timeout waiting for %s at %s: %w", label, addr, ctx.Err())
		}
		attempt++
		if verbose && out != nil {
			fmt.Fprintf(out, "seed: checking %s at %s (attempt %d)\n", label, addr, attempt)
		}
		dialCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
		cancel()
		if err == nil {
			_ = conn.Close()
			if verbose && out != nil {
				fmt.Fprintf(out, "seed: connected to %s at %s\n", label, addr)
			}
			return nil
		}
		if verbose && out != nil {
			fmt.Fprintf(out, "seed: %s not ready at %s: %v\n", label, addr, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
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
