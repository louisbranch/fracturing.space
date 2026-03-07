// Package notifications parses notifications command flags and launches the service.
package notifications

import (
	"context"
	"flag"
	"fmt"
	"strings"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/notifications/app"
	"golang.org/x/sync/errgroup"
)

type runtimeMode string

const (
	runtimeModeAPI    runtimeMode = "api"
	runtimeModeWorker runtimeMode = "worker"
	runtimeModeAll    runtimeMode = "all"
)

var (
	runWithTelemetry       = entrypoint.RunWithTelemetry
	startStatusReporter    = entrypoint.StartStatusReporter
	runNotificationsAPI    = server.Run
	runNotificationsWorker = server.RunEmailDeliveryWorker
)

// Config holds notifications command configuration.
type Config struct {
	Port       int    `env:"FRACTURING_SPACE_NOTIFICATIONS_PORT" envDefault:"8088"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
	Mode       string `env:"FRACTURING_SPACE_NOTIFICATIONS_MODE" envDefault:"api"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The notifications gRPC server port")
	fs.StringVar(&cfg.Mode, "mode", cfg.Mode, "Runtime mode: api, worker, or all")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	if _, err := parseRuntimeMode(cfg.Mode); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the notifications API service.
func Run(ctx context.Context, cfg Config) error {
	mode, err := parseRuntimeMode(cfg.Mode)
	if err != nil {
		return err
	}

	return runWithTelemetry(ctx, entrypoint.ServiceNotifications, func(context.Context) error {
		stopReporter := startStatusReporter(
			ctx,
			"notifications",
			cfg.StatusAddr,
			statusCapabilities(mode)...,
		)
		defer stopReporter()

		return runMode(ctx, cfg, mode)
	})
}

func parseRuntimeMode(value string) (runtimeMode, error) {
	switch runtimeMode(strings.ToLower(strings.TrimSpace(value))) {
	case runtimeModeAPI:
		return runtimeModeAPI, nil
	case runtimeModeWorker:
		return runtimeModeWorker, nil
	case runtimeModeAll:
		return runtimeModeAll, nil
	default:
		return "", fmt.Errorf("invalid notifications runtime mode %q (expected api, worker, or all)", value)
	}
}

func statusCapabilities(mode runtimeMode) []entrypoint.CapabilityRegistration {
	capabilities := make([]entrypoint.CapabilityRegistration, 0, 2)
	switch mode {
	case runtimeModeAPI:
		capabilities = append(capabilities, entrypoint.Capability("notifications.inbox", platformstatus.Operational))
	case runtimeModeWorker:
		capabilities = append(capabilities, entrypoint.Capability("notifications.email.delivery-worker", platformstatus.Operational))
	case runtimeModeAll:
		capabilities = append(
			capabilities,
			entrypoint.Capability("notifications.inbox", platformstatus.Operational),
			entrypoint.Capability("notifications.email.delivery-worker", platformstatus.Operational),
		)
	}
	return capabilities
}

func runMode(ctx context.Context, cfg Config, mode runtimeMode) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if mode == runtimeModeAPI {
		return runNotificationsAPI(ctx, cfg.Port)
	}
	if mode == runtimeModeWorker {
		return runNotificationsWorker(ctx)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return runNotificationsAPI(groupCtx, cfg.Port)
	})
	group.Go(func() error {
		return runNotificationsWorker(groupCtx)
	})
	return group.Wait()
}
