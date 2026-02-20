// Package web parses web command flags and boots the browser UI service.
package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/otel"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// Config holds the web command configuration.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR"           envDefault:"localhost:8086"`
	AuthBaseURL         string        `env:"FRACTURING_SPACE_WEB_AUTH_BASE_URL"       envDefault:"http://localhost:8084"`
	AuthAddr            string        `env:"FRACTURING_SPACE_WEB_AUTH_ADDR"           envDefault:"localhost:8083"`
	GameAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"              envDefault:"localhost:8080"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT"        envDefault:"2s"`
	OAuthClientID       string        `env:"FRACTURING_SPACE_WEB_OAUTH_CLIENT_ID"     envDefault:"fracturing-space-web"`
	CallbackURL         string        `env:"FRACTURING_SPACE_WEB_CALLBACK_URL"`
	AuthTokenURL        string        `env:"FRACTURING_SPACE_WEB_AUTH_TOKEN_URL"`
	Domain              string        `env:"FRACTURING_SPACE_DOMAIN"`
	OAuthResourceSecret string        `env:"FRACTURING_SPACE_WEB_OAUTH_RESOURCE_SECRET"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.AuthBaseURL, "auth-base-url", cfg.AuthBaseURL, "Auth service HTTP base URL")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Run builds and starts the web login surface.
func Run(ctx context.Context, cfg Config) error {
	shutdown, err := otel.Setup(ctx, "web")
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("otel shutdown: %v", err)
		}
	}()

	server, err := web.NewServer(web.Config{
		HTTPAddr:            cfg.HTTPAddr,
		AuthBaseURL:         cfg.AuthBaseURL,
		AuthAddr:            cfg.AuthAddr,
		GameAddr:            cfg.GameAddr,
		GRPCDialTimeout:     cfg.GRPCDialTimeout,
		OAuthClientID:       cfg.OAuthClientID,
		CallbackURL:         cfg.CallbackURL,
		AuthTokenURL:        cfg.AuthTokenURL,
		Domain:              cfg.Domain,
		OAuthResourceSecret: cfg.OAuthResourceSecret,
	})
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}
	defer server.Close()

	if err := server.ListenAndServe(ctx); err != nil {
		return fmt.Errorf("serve web: %w", err)
	}
	return nil
}
