// Package play parses play command flags and composes transport entrypoints.
package play

import (
	"context"
	"flag"
	"fmt"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	playapp "github.com/louisbranch/fracturing.space/internal/services/play/app"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Config holds play command configuration.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_PLAY_HTTP_ADDR" envDefault:":8094"`
	WebHTTPAddr         string `env:"FRACTURING_SPACE_WEB_HTTP_ADDR"`
	AuthAddr            string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	GameAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"`
	StatusAddr          string `env:"FRACTURING_SPACE_STATUS_ADDR"`
	DBPath              string `env:"FRACTURING_SPACE_PLAY_DB_PATH" envDefault:"data/play.db"`
	PlayUIDevServerURL  string `env:"FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL"`
	TrustForwardedProto bool   `env:"FRACTURING_SPACE_PLAY_TRUST_FORWARDED_PROTO" envDefault:"false"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.WebHTTPAddr = serviceaddr.OrDefaultHTTPAddr(cfg.WebHTTPAddr, serviceaddr.ServiceWeb)
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "play HTTP listen address")
	fs.StringVar(&cfg.WebHTTPAddr, "web-http-addr", cfg.WebHTTPAddr, "web HTTP listen address for browser fallback links")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "game service gRPC address")
	fs.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "play SQLite database path")
	fs.StringVar(&cfg.PlayUIDevServerURL, "ui-dev-server-url", cfg.PlayUIDevServerURL, "optional play UI dev server URL")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "trust X-Forwarded-Proto when resolving request scheme")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run builds the play app and starts browser-facing runtime behavior.
func Run(ctx context.Context, cfg Config) error {
	launchGrantCfg, err := playlaunchgrant.LoadConfigFromEnv(nil)
	if err != nil {
		return fmt.Errorf("load play launch grant config: %w", err)
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServicePlay, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"play",
			cfg.StatusAddr,
			entrypoint.Capability("play.http", platformstatus.Operational),
			entrypoint.Capability("play.game.integration", platformstatus.Operational),
			entrypoint.Capability("play.auth.integration", platformstatus.Operational),
		)
		defer stopReporter()

		server, err := playapp.NewServer(ctx, playapp.Config{
			HTTPAddr:            cfg.HTTPAddr,
			WebHTTPAddr:         cfg.WebHTTPAddr,
			AuthAddr:            cfg.AuthAddr,
			GameAddr:            cfg.GameAddr,
			DBPath:              cfg.DBPath,
			PlayUIDevServerURL:  cfg.PlayUIDevServerURL,
			RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
			LaunchGrant:         launchGrantCfg,
		})
		if err != nil {
			return fmt.Errorf("init play server: %w", err)
		}
		defer server.Close()
		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve play: %w", err)
		}
		return nil
	})
}
