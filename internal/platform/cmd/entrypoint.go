package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/otel"
)

const defaultOTelShutdownTimeout = 5 * time.Second

// Service identifiers for command startup telemetry and CLI naming consistency.
const (
	ServiceAdmin    = "admin"
	ServiceAI       = "ai"
	ServiceAuth     = "auth"
	ServiceChat     = "chat"
	ServiceGame     = "game"
	ServiceMCP      = "mcp"
	ServiceScenario = "scenario"
	ServiceSeed     = "seed"
	ServiceWeb      = "web"
)

// RunOptions controls shared entrypoint behavior for service commands.
type RunOptions struct {
	// ShutdownTimeout sets the timeout used when stopping telemetry.
	ShutdownTimeout time.Duration
}

// ParseConfig loads environment defaults into cfg.
func ParseConfig[T any](cfg *T) error {
	if cfg == nil {
		return errors.New("config target is required")
	}
	return config.ParseEnv(cfg)
}

// ParseArgs parses command-line flags.
func ParseArgs(fs *flag.FlagSet, args []string) error {
	if fs == nil {
		return errors.New("flag parser is required")
	}
	if args == nil {
		args = []string{}
	}
	return fs.Parse(args)
}

// ParseConfigFromArgs loads defaults from env and then parses flags.
func ParseConfigFromArgs[T any](cfg *T, fs *flag.FlagSet, args []string) error {
	if err := ParseConfig(cfg); err != nil {
		return err
	}
	return ParseArgs(fs, args)
}

// RunWithTelemetry configures observability and executes a service run loop.
func RunWithTelemetry(ctx context.Context, service string, run func(context.Context) error) error {
	return RunWithTelemetryAndOptions(ctx, service, RunOptions{}, run)
}

// RunWithTelemetryAndOptions configures observability and executes a service run loop.
func RunWithTelemetryAndOptions(ctx context.Context, service string, options RunOptions, run func(context.Context) error) error {
	service = strings.TrimSpace(service)
	if service == "" {
		return fmt.Errorf("service name is required")
	}
	if run == nil {
		return fmt.Errorf("run function is required")
	}
	shutdown, err := otel.Setup(ctx, service)
	if err != nil {
		return err
	}
	defer func() {
		shutdownTimeout := options.ShutdownTimeout
		if shutdownTimeout <= 0 {
			shutdownTimeout = defaultOTelShutdownTimeout
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("%s otel shutdown: %v", service, err)
		}
	}()
	return run(ctx)
}
