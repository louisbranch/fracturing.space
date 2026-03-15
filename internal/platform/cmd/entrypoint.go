package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/otel"
)

const defaultOTelShutdownTimeout = 5 * time.Second

// Service identifiers for command startup telemetry and CLI naming consistency.
const (
	ServiceAdmin         = "admin"
	ServiceAI            = "ai"
	ServiceSocial        = "social"
	ServiceDiscovery     = "discovery"
	ServiceAuth          = "auth"
	ServiceGame          = "game"
	ServiceMCP           = "mcp"
	ServicePlay          = "play"
	ServiceNotifications = "notifications"
	ServiceScenario      = "scenario"
	ServiceSeed          = "seed"
	ServiceStatus        = "status"
	ServiceUserHub       = "userhub"
	ServiceWeb           = "web"
	ServiceWorker        = "worker"
)

// RunOptions controls shared entrypoint behavior for service commands.
type RunOptions struct {
	// ShutdownTimeout sets the timeout used when stopping telemetry.
	ShutdownTimeout time.Duration
}

// ServiceMainOptions configures shared process startup for long-running services.
//
// This keeps `cmd/*/main.go` thin and consistent across services while retaining
// an explicit contract for configuration parsing and runtime execution.
type ServiceMainOptions[T any] struct {
	// Service is the canonical service identifier used for log prefix and errors.
	Service string
	// ParseConfig maps args/env into a runtime config for the service.
	ParseConfig func(fs *flag.FlagSet, args []string) (T, error)
	// Run starts the service and blocks until shutdown.
	Run func(ctx context.Context, cfg T) error
	// FlagSet is optional; defaults to flag.CommandLine.
	FlagSet *flag.FlagSet
	// Args is optional; defaults to os.Args[1:].
	Args []string
	// BaseContext is optional; defaults to context.Background().
	BaseContext context.Context
	// Signals is optional; defaults to os.Interrupt and syscall.SIGTERM.
	Signals []os.Signal
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

// RunServiceMain standardizes process bootstrap for long-running service
// binaries.
func RunServiceMain[T any](options ServiceMainOptions[T]) error {
	service := strings.TrimSpace(options.Service)
	if service == "" {
		return errors.New("service name is required")
	}
	if options.ParseConfig == nil {
		return errors.New("parse config function is required")
	}
	if options.Run == nil {
		return errors.New("run function is required")
	}

	fs := options.FlagSet
	if fs == nil {
		fs = flag.CommandLine
	}
	args := options.Args
	if args == nil {
		args = os.Args[1:]
	}
	baseCtx := options.BaseContext
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	signals := options.Signals
	if len(signals) == 0 {
		signals = []os.Signal{os.Interrupt, syscall.SIGTERM}
	}

	log.SetPrefix(fmt.Sprintf("[%s] ", strings.ToUpper(service)))

	cfg, err := options.ParseConfig(fs, args)
	if err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	ctx, stop := signal.NotifyContext(baseCtx, signals...)
	defer stop()

	if err := options.Run(ctx, cfg); err != nil {
		return fmt.Errorf("serve %s: %w", service, err)
	}

	return nil
}
