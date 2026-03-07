// Package userhub parses userhub command flags and launches the userhub runtime.
package userhub

import (
	"context"
	"flag"
	"time"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	userhubserver "github.com/louisbranch/fracturing.space/internal/services/userhub/app"
)

// Config holds userhub command configuration.
type Config struct {
	Port              int           `env:"FRACTURING_SPACE_USERHUB_PORT" envDefault:"8092"`
	GameAddr          string        `env:"FRACTURING_SPACE_USERHUB_GAME_ADDR"`
	SocialAddr        string        `env:"FRACTURING_SPACE_USERHUB_SOCIAL_ADDR"`
	NotificationsAddr string        `env:"FRACTURING_SPACE_USERHUB_NOTIFICATIONS_ADDR"`
	StatusAddr        string        `env:"FRACTURING_SPACE_USERHUB_STATUS_ADDR"`
	CacheFreshTTL     time.Duration `env:"FRACTURING_SPACE_USERHUB_CACHE_FRESH_TTL" envDefault:"15s"`
	CacheStaleTTL     time.Duration `env:"FRACTURING_SPACE_USERHUB_CACHE_STALE_TTL" envDefault:"2m"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.SocialAddr = serviceaddr.OrDefaultGRPCAddr(cfg.SocialAddr, serviceaddr.ServiceSocial)
	cfg.NotificationsAddr = serviceaddr.OrDefaultGRPCAddr(cfg.NotificationsAddr, serviceaddr.ServiceNotifications)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.IntVar(&cfg.Port, "port", cfg.Port, "The userhub gRPC server port")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "The game gRPC server address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "The social gRPC server address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "The notifications gRPC server address")
	fs.DurationVar(&cfg.CacheFreshTTL, "cache-fresh-ttl", cfg.CacheFreshTTL, "The fresh dashboard cache TTL")
	fs.DurationVar(&cfg.CacheStaleTTL, "cache-stale-ttl", cfg.CacheStaleTTL, "The stale dashboard fallback TTL")

	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the userhub runtime.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceUserHub, func(context.Context) error {
		return userhubserver.Run(ctx, userhubserver.RuntimeConfig{
			Port:              cfg.Port,
			GameAddr:          cfg.GameAddr,
			SocialAddr:        cfg.SocialAddr,
			NotificationsAddr: cfg.NotificationsAddr,
			StatusAddr:        cfg.StatusAddr,
			CacheFreshTTL:     cfg.CacheFreshTTL,
			CacheStaleTTL:     cfg.CacheStaleTTL,
		})
	})
}
