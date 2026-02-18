// Package scenario parses scenario command flags and executes scripted runs.
package scenario

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
)

// Config holds scenario command configuration.
type Config struct {
	GRPCAddr         string        `env:"FRACTURING_SPACE_GAME_ADDR"               envDefault:"localhost:8080"`
	Scenario         string        `env:"FRACTURING_SPACE_SCENARIO_FILE"`
	Assertions       bool          `env:"FRACTURING_SPACE_SCENARIO_ASSERT"         envDefault:"true"`
	Verbose          bool          `env:"FRACTURING_SPACE_SCENARIO_VERBOSE"`
	Timeout          time.Duration `env:"FRACTURING_SPACE_SCENARIO_TIMEOUT"        envDefault:"10s"`
	ValidateComments bool          `env:"FRACTURING_SPACE_SCENARIO_VALIDATE_COMMENTS" envDefault:"true"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}

	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.Scenario, "scenario", cfg.Scenario, "path to scenario lua file")
	fs.BoolVar(&cfg.Assertions, "assert", cfg.Assertions, "enable assertions (disable to log expectations)")
	fs.BoolVar(&cfg.ValidateComments, "validate-comments", cfg.ValidateComments, "require each scene block to start with a comment")
	fs.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "enable verbose logging")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "timeout per step")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run executes a scenario Lua file through game gRPC contracts.
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
		GRPCAddr:         cfg.GRPCAddr,
		Timeout:          cfg.Timeout,
		Assertions:       mode,
		Verbose:          cfg.Verbose,
		Logger:           logger,
		ValidateComments: cfg.ValidateComments,
	}, cfg.Scenario)
}
