// Package chat parses chat command flags and composes transport entrypoints.
package chat

import (
	"context"
	"flag"
	"fmt"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/chat/app"
)

// Config holds chat command configuration.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR"       envDefault:":8086"`
	AuthAddr            string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	GameAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr              string `env:"FRACTURING_SPACE_AI_ADDR"`
	AuthBaseURL         string `env:"FRACTURING_SPACE_WEB_AUTH_BASE_URL"    envDefault:"http://localhost:8084"`
	OAuthResourceSecret string `env:"FRACTURING_SPACE_WEB_OAUTH_RESOURCE_SECRET"`
	StatusAddr          string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.AIAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AIAddr, serviceaddr.ServiceAI)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "ai service gRPC address")
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
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"chat",
			cfg.StatusAddr,
			entrypoint.Capability("chat.realtime", platformstatus.Operational),
			entrypoint.Capability("chat.game.integration", platformstatus.Operational),
			entrypoint.Capability("chat.ai.integration", platformstatus.Operational),
			entrypoint.Capability("chat.auth.integration", platformstatus.Operational),
		)
		defer stopReporter()

		if err := server.Run(ctx, server.Config{
			HTTPAddr:            cfg.HTTPAddr,
			AuthAddr:            cfg.AuthAddr,
			GameAddr:            cfg.GameAddr,
			AIAddr:              cfg.AIAddr,
			AuthBaseURL:         cfg.AuthBaseURL,
			OAuthResourceSecret: cfg.OAuthResourceSecret,
		}); err != nil {
			return fmt.Errorf("serve chat: %w", err)
		}
		return nil
	})
}
