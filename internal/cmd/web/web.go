package web

import (
	"context"
	"flag"
	"fmt"
	"log/slog"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Config holds command inputs for web startup.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_WEB_HTTP_ADDR" envDefault:"localhost:8080"`
	PlayHTTPAddr        string `env:"FRACTURING_SPACE_PLAY_HTTP_ADDR" envDefault:"localhost:8094"`
	TrustForwardedProto bool   `env:"FRACTURING_SPACE_WEB_TRUST_FORWARDED_PROTO" envDefault:"false"`
	AuthAddr            string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr          string `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	GameAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"`
	InviteAddr          string `env:"FRACTURING_SPACE_INVITE_ADDR"`
	AIAddr              string `env:"FRACTURING_SPACE_AI_ADDR"`
	DiscoveryAddr       string `env:"FRACTURING_SPACE_DISCOVERY_ADDR"`
	NotificationsAddr   string `env:"FRACTURING_SPACE_NOTIFICATIONS_ADDR"`
	UserHubAddr         string `env:"FRACTURING_SPACE_USERHUB_ADDR"`
	StatusAddr          string `env:"FRACTURING_SPACE_STATUS_ADDR"`
	AssetBaseURL        string `env:"FRACTURING_SPACE_ASSET_BASE_URL"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	if err := validateDependencyAddressBindingsCoverage(); err != nil {
		return Config{}, err
	}
	applyDependencyAddressDefaults(&cfg)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.PlayHTTPAddr, "play-http-addr", cfg.PlayHTTPAddr, "Play HTTP listen address")
	applyDependencyAddressFlags(fs, &cfg)
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "Trust X-Forwarded-Proto when resolving request scheme")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the web service process.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWeb, func(context.Context) error {
		reporter := platformstatus.NewReporter("web", nil)
		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		runtimeDeps, err := bootstrapRuntimeDependencies(ctx, cfg, reporter, nil)
		if err != nil {
			return err
		}
		defer runtimeDeps.close()
		playLaunchGrantCfg, err := playlaunchgrant.LoadConfigFromEnv(nil)
		if err != nil {
			return fmt.Errorf("load play launch grant config: %w", err)
		}

		server, err := web.NewServer(ctx, cfg.serverConfig(runtimeDeps.bundle, playLaunchGrantCfg))
		if err != nil {
			return fmt.Errorf("init web server: %w", err)
		}
		defer server.Close()
		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve web: %w", err)
		}
		return nil
	})
}

// serverConfig maps command config and assembled runtime dependencies into the
// web server constructor contract.
func (cfg Config) serverConfig(deps web.DependencyBundle, playLaunchGrant playlaunchgrant.Config) web.Config {
	return web.Config{
		HTTPAddr:            cfg.HTTPAddr,
		PlayHTTPAddr:        cfg.PlayHTTPAddr,
		Logger:              slog.Default(),
		PlayLaunchGrant:     playLaunchGrant,
		RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
		Dependencies:        &deps,
	}
}
