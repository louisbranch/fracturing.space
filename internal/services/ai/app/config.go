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
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// serverEnv captures startup configuration and optional provider integration.
type serverEnv struct {
	DBPath                   string `env:"FRACTURING_SPACE_AI_DB_PATH"`
	EncryptionKey            string `env:"FRACTURING_SPACE_AI_ENCRYPTION_KEY"`
	GameAddr                 string `env:"FRACTURING_SPACE_GAME_ADDR"`
	InternalServiceAllowlist string `env:"FRACTURING_SPACE_AI_INTERNAL_SERVICE_ALLOWLIST" envDefault:"ai,worker,game"`

	OpenAIOAuthAuthURL       string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL"`
	OpenAIOAuthTokenURL      string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL"`
	OpenAIOAuthClientID      string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID"`
	OpenAIOAuthClientSecret  string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET"`
	OpenAIOAuthRedirectURI   string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI"`
	OpenAIResponsesURL       string        `env:"FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL"`
	OrchestrationTurnTimeout time.Duration `env:"FRACTURING_SPACE_AI_ORCHESTRATION_TURN_TIMEOUT" envDefault:"2m"`
	OrchestrationMaxSteps    int           `env:"FRACTURING_SPACE_AI_ORCHESTRATION_MAX_STEPS" envDefault:"8"`
	ToolResultMaxBytes       int           `env:"FRACTURING_SPACE_AI_ORCHESTRATION_TOOL_RESULT_MAX_BYTES" envDefault:"32768"`
	DaggerheartReferenceRoot string        `env:"FRACTURING_SPACE_AI_DAGGERHEART_REFERENCE_ROOT"`
	InstructionsRoot         string        `env:"FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT"`
}

// runtimeConfig is the normalized startup configuration used by the AI runtime.
type runtimeConfig struct {
	DBPath                   string
	EncryptionKey            string
	GameAddr                 string
	InternalServiceAllowlist map[string]struct{}
	OpenAIOAuthConfig        *openaiprovider.OAuthConfig
	OpenAIResponsesURL       string
	OrchestrationTurnTimeout time.Duration
	OrchestrationMaxSteps    int
	ToolResultMaxBytes       int
	DaggerheartReferenceRoot string
	InstructionsRoot         string
	SessionGrantConfig       *aisessiongrant.Config
}

// Validate rejects incomplete runtime configuration before server
// construction allocates listeners, stores, or client connections.
func (cfg runtimeConfig) Validate() error {
	if strings.TrimSpace(cfg.EncryptionKey) == "" {
		return fmt.Errorf("FRACTURING_SPACE_AI_ENCRYPTION_KEY is required")
	}
	if _, err := decodeBase64Key(cfg.EncryptionKey); err != nil {
		return fmt.Errorf("decode encryption key: %w", err)
	}
	if cfg.OrchestrationTurnTimeout <= 0 {
		return fmt.Errorf("FRACTURING_SPACE_AI_ORCHESTRATION_TURN_TIMEOUT must be positive")
	}
	if cfg.OrchestrationMaxSteps <= 0 {
		return fmt.Errorf("FRACTURING_SPACE_AI_ORCHESTRATION_MAX_STEPS must be positive")
	}
	if cfg.ToolResultMaxBytes <= 0 {
		return fmt.Errorf("FRACTURING_SPACE_AI_ORCHESTRATION_TOOL_RESULT_MAX_BYTES must be positive")
	}
	return nil
}

func loadServerEnv() (serverEnv, error) {
	var cfg serverEnv
	if err := config.ParseEnv(&cfg); err != nil {
		return serverEnv{}, err
	}
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "ai.db")
	}
	return cfg, nil
}

// loadRuntimeConfigFromEnv parses and validates AI runtime startup config once
// so server construction has one deterministic config source.
func loadRuntimeConfigFromEnv() (runtimeConfig, error) {
	srvEnv, err := loadServerEnv()
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("load AI runtime env: %w", err)
	}
	openAIOAuthConfig, err := openAIOAuthConfig(srvEnv)
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("load OpenAI OAuth config: %w", err)
	}
	sessionGrantConfig, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		return runtimeConfig{}, fmt.Errorf("load AI session grant config: %w", err)
	}

	cfg := runtimeConfig{
		DBPath:                   strings.TrimSpace(srvEnv.DBPath),
		EncryptionKey:            strings.TrimSpace(srvEnv.EncryptionKey),
		GameAddr:                 strings.TrimSpace(srvEnv.GameAddr),
		InternalServiceAllowlist: parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
		OpenAIOAuthConfig:        openAIOAuthConfig,
		OpenAIResponsesURL:       strings.TrimSpace(srvEnv.OpenAIResponsesURL),
		OrchestrationTurnTimeout: srvEnv.OrchestrationTurnTimeout,
		OrchestrationMaxSteps:    srvEnv.OrchestrationMaxSteps,
		ToolResultMaxBytes:       srvEnv.ToolResultMaxBytes,
		DaggerheartReferenceRoot: strings.TrimSpace(srvEnv.DaggerheartReferenceRoot),
		InstructionsRoot:         strings.TrimSpace(srvEnv.InstructionsRoot),
		SessionGrantConfig:       sessionGrantConfig,
	}
	if err := cfg.Validate(); err != nil {
		return runtimeConfig{}, err
	}
	return cfg, nil
}

func (cfg runtimeConfig) campaignTurnRunnerConfig(dialer orchestration.Dialer) orchestration.RunnerConfig {
	return orchestration.RunnerConfig{
		Dialer:             dialer,
		MaxSteps:           cfg.OrchestrationMaxSteps,
		TurnTimeout:        cfg.OrchestrationTurnTimeout,
		ToolResultMaxBytes: cfg.ToolResultMaxBytes,
	}
}

// openAIOAuthConfigFromEnv loads optional OpenAI OAuth config.
//
// If all OpenAI OAuth variables are present they are wired in together; partial
// configuration is rejected to avoid accidental half-configured runtime.
func openAIOAuthConfigFromEnv() (*openaiprovider.OAuthConfig, error) {
	env, err := loadServerEnv()
	if err != nil {
		return nil, fmt.Errorf("load AI runtime env: %w", err)
	}
	return openAIOAuthConfig(env)
}

func openAIOAuthConfig(env serverEnv) (*openaiprovider.OAuthConfig, error) {
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
	return &openaiprovider.OAuthConfig{
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
