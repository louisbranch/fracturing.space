// Package chat parses chat command flags and composes transport entrypoints.
package chat

import (
	"context"
	"flag"
	"fmt"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	server "github.com/louisbranch/fracturing.space/internal/services/chat/app"
)

// Config holds chat command configuration.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR"       envDefault:":8086"`
	GameAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"            envDefault:"localhost:8082"`
	AuthBaseURL         string `env:"FRACTURING_SPACE_WEB_AUTH_BASE_URL"    envDefault:"http://localhost:8084"`
	OAuthResourceSecret string `env:"FRACTURING_SPACE_WEB_OAUTH_RESOURCE_SECRET"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "chat HTTP listen address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "game service gRPC address")
	fs.StringVar(&cfg.AuthBaseURL, "auth-base-url", cfg.AuthBaseURL, "auth service base URL")
	fs.StringVar(&cfg.OAuthResourceSecret, "oauth-resource-secret", cfg.OAuthResourceSecret, "auth introspection resource secret")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run builds the chat app and starts realtime transport behavior.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceChat, func(context.Context) error {
		if err := server.Run(ctx, server.Config{
			HTTPAddr:            cfg.HTTPAddr,
			GameAddr:            cfg.GameAddr,
			AuthBaseURL:         cfg.AuthBaseURL,
			OAuthResourceSecret: cfg.OAuthResourceSecret,
		}); err != nil {
			return fmt.Errorf("serve chat: %w", err)
		}
		return nil
	})
}
