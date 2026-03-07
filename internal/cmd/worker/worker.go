// Package worker parses worker command flags and launches the worker runtime.
package worker

import (
	"context"
	"flag"
	"log"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	workerserver "github.com/louisbranch/fracturing.space/internal/services/worker/app"
)

// Config holds worker command configuration.
type Config struct {
	Port              int           `env:"FRACTURING_SPACE_WORKER_PORT" envDefault:"8089"`
	AuthAddr          string        `env:"FRACTURING_SPACE_WORKER_AUTH_ADDR"`
	SocialAddr        string        `env:"FRACTURING_SPACE_WORKER_SOCIAL_ADDR"`
	NotificationsAddr string        `env:"FRACTURING_SPACE_WORKER_NOTIFICATIONS_ADDR"`
	DBPath            string        `env:"FRACTURING_SPACE_WORKER_DB_PATH" envDefault:"data/worker.db"`
	Consumer          string        `env:"FRACTURING_SPACE_WORKER_CONSUMER" envDefault:"worker-onboarding"`
	PollInterval      time.Duration `env:"FRACTURING_SPACE_WORKER_POLL_INTERVAL" envDefault:"2s"`
	LeaseTTL          time.Duration `env:"FRACTURING_SPACE_WORKER_LEASE_TTL" envDefault:"30s"`
	MaxAttempts       int           `env:"FRACTURING_SPACE_WORKER_MAX_ATTEMPTS" envDefault:"8"`
	RetryBackoff      time.Duration `env:"FRACTURING_SPACE_WORKER_RETRY_BACKOFF" envDefault:"5s"`
	RetryMaxDelay     time.Duration `env:"FRACTURING_SPACE_WORKER_RETRY_MAX_DELAY" envDefault:"5m"`
	GRPCDialTimeout   time.Duration `env:"FRACTURING_SPACE_WORKER_DIAL_TIMEOUT" envDefault:"2s"`
	StatusAddr        string        `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.SocialAddr = serviceaddr.OrDefaultGRPCAddr(cfg.SocialAddr, serviceaddr.ServiceSocial)
	cfg.NotificationsAddr = serviceaddr.OrDefaultGRPCAddr(cfg.NotificationsAddr, serviceaddr.ServiceNotifications)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The worker health gRPC server port")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "The auth gRPC server address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "The social gRPC server address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "The notifications gRPC server address")
	fs.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "The worker SQLite database path")
	fs.StringVar(&cfg.Consumer, "consumer", cfg.Consumer, "Integration outbox consumer name")
	fs.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "Integration outbox poll interval")
	fs.DurationVar(&cfg.LeaseTTL, "lease-ttl", cfg.LeaseTTL, "Integration outbox lease duration")
	fs.IntVar(&cfg.MaxAttempts, "max-attempts", cfg.MaxAttempts, "Maximum processing attempts before dead-letter")
	fs.DurationVar(&cfg.RetryBackoff, "retry-backoff", cfg.RetryBackoff, "Base retry backoff delay")
	fs.DurationVar(&cfg.RetryMaxDelay, "retry-max-delay", cfg.RetryMaxDelay, "Maximum retry delay")
	fs.DurationVar(&cfg.GRPCDialTimeout, "dial-timeout", cfg.GRPCDialTimeout, "gRPC dependency dial timeout")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the worker runtime.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWorker, func(context.Context) error {
		// Status reporter.
		statusConn := platformgrpc.DialLenient(ctx, cfg.StatusAddr, log.Printf)
		if statusConn != nil {
			defer func() {
				if err := statusConn.Close(); err != nil {
					log.Printf("close status connection: %v", err)
				}
			}()
		}
		var statusClient statusv1.StatusServiceClient
		if statusConn != nil {
			statusClient = statusv1.NewStatusServiceClient(statusConn)
		}
		reporter := platformstatus.NewReporter("worker", statusClient)
		reporter.Register("worker.processing", platformstatus.Operational)
		reporter.Register("worker.auth.integration", platformstatus.Operational)
		reporter.Register("worker.social.integration", platformstatus.Operational)
		reporter.Register("worker.notifications.integration", platformstatus.Operational)
		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		return workerserver.Run(ctx, workerserver.RuntimeConfig{
			Port:              cfg.Port,
			AuthAddr:          cfg.AuthAddr,
			SocialAddr:        cfg.SocialAddr,
			NotificationsAddr: cfg.NotificationsAddr,
			DBPath:            cfg.DBPath,
			Consumer:          cfg.Consumer,
			PollInterval:      cfg.PollInterval,
			LeaseTTL:          cfg.LeaseTTL,
			MaxAttempts:       cfg.MaxAttempts,
			RetryBackoff:      cfg.RetryBackoff,
			RetryMaxDelay:     cfg.RetryMaxDelay,
			GRPCDialTimeout:   cfg.GRPCDialTimeout,
		})
	})
}
