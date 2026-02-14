package scenario

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
)

// Config holds scenario command configuration.
type Config struct {
	GRPCAddr   string        `env:"FRACTURING_SPACE_GAME_ADDR"         envDefault:"localhost:8080"`
	Scenario   string        `env:"FRACTURING_SPACE_SCENARIO_FILE"`
	Assertions bool          `env:"FRACTURING_SPACE_SCENARIO_ASSERT"   envDefault:"true"`
	Verbose    bool          `env:"FRACTURING_SPACE_SCENARIO_VERBOSE"`
	Timeout    time.Duration `env:"FRACTURING_SPACE_SCENARIO_TIMEOUT"  envDefault:"10s"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.Scenario, "scenario", cfg.Scenario, "path to scenario lua file")
	fs.BoolVar(&cfg.Assertions, "assert", cfg.Assertions, "enable assertions (disable to log expectations)")
	fs.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "enable verbose logging")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "timeout per step")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run executes the scenario command.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if cfg.Scenario == "" {
		return errors.New("scenario path is required")
	}

	mode := scenario.AssertionStrict
	if !cfg.Assertions {
		mode = scenario.AssertionLogOnly
	}

	logger := log.New(errOut, "", 0)
	return scenario.RunFile(ctx, scenario.Config{
		GRPCAddr:   cfg.GRPCAddr,
		Timeout:    cfg.Timeout,
		Assertions: mode,
		Verbose:    cfg.Verbose,
		Logger:     logger,
	}, cfg.Scenario)
}
