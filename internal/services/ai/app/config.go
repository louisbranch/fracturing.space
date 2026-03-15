package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// serverEnv captures startup configuration and optional provider integration.
type serverEnv struct {
	DBPath                   string `env:"FRACTURING_SPACE_AI_DB_PATH"`
	EncryptionKey            string `env:"FRACTURING_SPACE_AI_ENCRYPTION_KEY"`
	GameAddr                 string `env:"FRACTURING_SPACE_GAME_ADDR"`
	InternalServiceAllowlist string `env:"FRACTURING_SPACE_AI_INTERNAL_SERVICE_ALLOWLIST" envDefault:"ai,mcp,worker,game"`

	OpenAIOAuthAuthURL       string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL"`
	OpenAIOAuthTokenURL      string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL"`
	OpenAIOAuthClientID      string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID"`
	OpenAIOAuthClientSecret  string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET"`
	OpenAIOAuthRedirectURI   string `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI"`
	OpenAIResponsesURL       string `env:"FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL"`
	MCPURL                   string `env:"FRACTURING_SPACE_AI_MCP_URL"`
	DaggerheartReferenceRoot string `env:"FRACTURING_SPACE_AI_DAGGERHEART_REFERENCE_ROOT"`
}

// runtimeConfig is the normalized startup configuration used by the AI runtime.
type runtimeConfig struct {
	DBPath                   string
	EncryptionKey            string
	GameAddr                 string
	InternalServiceAllowlist map[string]struct{}
	OpenAIOAuthConfig        *aiservice.OpenAIOAuthConfig
	OpenAIResponsesURL       string
	MCPURL                   string
	DaggerheartReferenceRoot string
	SessionGrantConfig       *aisessiongrant.Config
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "ai.db")
	}
	return cfg
}

// loadRuntimeConfigFromEnv parses and validates AI runtime startup config once
// so server construction has one deterministic config source.
func loadRuntimeConfigFromEnv() (runtimeConfig, error) {
	srvEnv := loadServerEnv()
	openAIOAuthConfig, err := openAIOAuthConfig(srvEnv)
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("load OpenAI OAuth config: %w", err)
	}
	sessionGrantConfig, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("load AI session grant config: %w", err)
	}

	return runtimeConfig{
		DBPath:                   strings.TrimSpace(srvEnv.DBPath),
		EncryptionKey:            strings.TrimSpace(srvEnv.EncryptionKey),
		GameAddr:                 strings.TrimSpace(srvEnv.GameAddr),
		InternalServiceAllowlist: parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
		OpenAIOAuthConfig:        openAIOAuthConfig,
		OpenAIResponsesURL:       strings.TrimSpace(srvEnv.OpenAIResponsesURL),
		MCPURL:                   strings.TrimSpace(srvEnv.MCPURL),
		DaggerheartReferenceRoot: strings.TrimSpace(srvEnv.DaggerheartReferenceRoot),
		SessionGrantConfig:       sessionGrantConfig,
	}, nil
}

// openAIOAuthConfigFromEnv loads optional OpenAI OAuth config.
//
// If all OpenAI OAuth variables are present they are wired in together; partial
// configuration is rejected to avoid accidental half-configured runtime.
func openAIOAuthConfigFromEnv() (*aiservice.OpenAIOAuthConfig, error) {
	return openAIOAuthConfig(loadServerEnv())
}

func openAIOAuthConfig(env serverEnv) (*aiservice.OpenAIOAuthConfig, error) {
	authURL := strings.TrimSpace(env.OpenAIOAuthAuthURL)
	tokenURL := strings.TrimSpace(env.OpenAIOAuthTokenURL)
	clientID := strings.TrimSpace(env.OpenAIOAuthClientID)
	clientSecret := strings.TrimSpace(env.OpenAIOAuthClientSecret)
	redirectURI := strings.TrimSpace(env.OpenAIOAuthRedirectURI)

	required := map[string]string{
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL":      authURL,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL":     tokenURL,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID":     clientID,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET": clientSecret,
		"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI":  redirectURI,
	}

	setCount := 0
	missing := make([]string, 0, len(required))
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
			continue
		}
		setCount++
	}
	if setCount == 0 {
		return nil, nil
	}
	if setCount != len(required) {
		sort.Strings(missing)
		return nil, fmt.Errorf("partial OpenAI OAuth env config; missing: %s", strings.Join(missing, ", "))
	}

	// Keep provider secrets in-memory only; callers must never log this struct.
	return &aiservice.OpenAIOAuthConfig{
		AuthorizationURL: authURL,
		TokenURL:         tokenURL,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		RedirectURI:      redirectURI,
	}, nil
}

func aiSessionGrantConfigFromEnv() (*aisessiongrant.Config, error) {
	keys := []string{
		"FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER",
		"FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE",
		"FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY",
		"FRACTURING_SPACE_AI_SESSION_GRANT_TTL",
	}
	set := false
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			continue
		}
		set = true
		break
	}
	if !set {
		return nil, nil
	}
	cfg, err := aisessiongrant.LoadConfigFromEnv(time.Now)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func parseInternalServiceAllowlist(raw string) map[string]struct{} {
	values := strings.Split(strings.TrimSpace(raw), ",")
	allowlist := make(map[string]struct{}, len(values))
	for _, value := range values {
		serviceID := strings.ToLower(strings.TrimSpace(value))
		if serviceID == "" {
			continue
		}
		allowlist[serviceID] = struct{}{}
	}
	return allowlist
}
