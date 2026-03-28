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
	"github.com/louisbranch/fracturing.space/internal/services/ai/openviking"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	openaiprovider "github.com/louisbranch/fracturing.space/internal/services/ai/provider/openai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// serverEnv captures startup configuration and optional provider integration.
type serverEnv struct {
	DBPath                   string `env:"FRACTURING_SPACE_AI_DB_PATH"`
	EncryptionKey            string `env:"FRACTURING_SPACE_AI_ENCRYPTION_KEY"`
	GameAddr                 string `env:"FRACTURING_SPACE_GAME_ADDR"`
	InternalServiceAllowlist string `env:"FRACTURING_SPACE_AI_INTERNAL_SERVICE_ALLOWLIST" envDefault:"ai,worker,game"`

	OpenAIOAuthAuthURL                   string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL"`
	OpenAIOAuthTokenURL                  string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL"`
	OpenAIOAuthClientID                  string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID"`
	OpenAIOAuthClientSecret              string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET"`
	OpenAIOAuthRedirectURI               string        `env:"FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI"`
	OpenAIResponsesURL                   string        `env:"FRACTURING_SPACE_AI_OPENAI_RESPONSES_URL"`
	AnthropicBaseURL                     string        `env:"FRACTURING_SPACE_AI_ANTHROPIC_BASE_URL"`
	OrchestrationTurnTimeout             time.Duration `env:"FRACTURING_SPACE_AI_ORCHESTRATION_TURN_TIMEOUT" envDefault:"2m"`
	OrchestrationMaxSteps                int           `env:"FRACTURING_SPACE_AI_ORCHESTRATION_MAX_STEPS" envDefault:"8"`
	ToolResultMaxBytes                   int           `env:"FRACTURING_SPACE_AI_ORCHESTRATION_TOOL_RESULT_MAX_BYTES" envDefault:"32768"`
	DaggerheartReferenceRoot             string        `env:"FRACTURING_SPACE_AI_DAGGERHEART_REFERENCE_ROOT"`
	InstructionsRoot                     string        `env:"FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT"`
	OpenVikingBaseURL                    string        `env:"FRACTURING_SPACE_AI_OPENVIKING_BASE_URL"`
	OpenVikingMode                       string        `env:"FRACTURING_SPACE_AI_OPENVIKING_MODE" envDefault:"legacy"`
	OpenVikingSessionSyncEnabled         bool          `env:"FRACTURING_SPACE_AI_OPENVIKING_SESSION_SYNC_ENABLED" envDefault:"true"`
	OpenVikingAPIKey                     string        `env:"FRACTURING_SPACE_AI_OPENVIKING_API_KEY"`
	OpenVikingTimeout                    time.Duration `env:"FRACTURING_SPACE_AI_OPENVIKING_TIMEOUT" envDefault:"15s"`
	OpenVikingMirrorRoot                 string        `env:"FRACTURING_SPACE_AI_OPENVIKING_MIRROR_ROOT"`
	OpenVikingVisibleMirrorRoot          string        `env:"FRACTURING_SPACE_AI_OPENVIKING_VISIBLE_MIRROR_ROOT"`
	OpenVikingReferenceCorpusRoot        string        `env:"FRACTURING_SPACE_AI_OPENVIKING_REFERENCE_CORPUS_ROOT"`
	OpenVikingReferenceCorpusVisibleRoot string        `env:"FRACTURING_SPACE_AI_OPENVIKING_REFERENCE_CORPUS_VISIBLE_ROOT"`
	OpenVikingMaxResults                 int           `env:"FRACTURING_SPACE_AI_OPENVIKING_MAX_RESULTS" envDefault:"4"`
	OpenVikingMaxSections                int           `env:"FRACTURING_SPACE_AI_OPENVIKING_MAX_SECTIONS" envDefault:"2"`
	OpenVikingMinRelevanceScore          float64       `env:"FRACTURING_SPACE_AI_OPENVIKING_MIN_RELEVANCE_SCORE" envDefault:"0"`
	OpenVikingResourceSync               time.Duration `env:"FRACTURING_SPACE_AI_OPENVIKING_RESOURCE_SYNC_TIMEOUT" envDefault:"2s"`
}

// runtimeConfig is the normalized startup configuration used by the AI runtime.
type runtimeConfig struct {
	DBPath                               string
	EncryptionKey                        string
	GameAddr                             string
	InternalServiceAllowlist             map[string]struct{}
	OpenAIOAuthConfig                    *openaiprovider.OAuthConfig
	OpenAIResponsesURL                   string
	AnthropicBaseURL                     string
	OrchestrationTurnTimeout             time.Duration
	OrchestrationMaxSteps                int
	ToolResultMaxBytes                   int
	DaggerheartReferenceRoot             string
	InstructionsRoot                     string
	OpenVikingBaseURL                    string
	OpenVikingMode                       string
	OpenVikingSessionSyncEnabled         bool
	OpenVikingAPIKey                     string
	OpenVikingTimeout                    time.Duration
	OpenVikingMirrorRoot                 string
	OpenVikingVisibleMirrorRoot          string
	OpenVikingReferenceCorpusRoot        string
	OpenVikingReferenceCorpusVisibleRoot string
	OpenVikingMaxResults                 int
	OpenVikingMaxSections                int
	OpenVikingMinRelevanceScore          float64
	OpenVikingResourceSync               time.Duration
	SessionGrantConfig                   *aisessiongrant.Config
}

// Validate rejects incomplete runtime configuration before server
// construction allocates listeners, stores, or client connections.
func (cfg runtimeConfig) Validate() error {
	if strings.TrimSpace(cfg.DBPath) == "" {
		return fmt.Errorf("FRACTURING_SPACE_AI_DB_PATH is required")
	}
	if _, err := cfg.encryptionKeyBytes(); err != nil {
		return err
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
	if cfg.OpenVikingBaseURL != "" {
		if _, err := openviking.ParseIntegrationMode(cfg.OpenVikingMode); err != nil {
			return err
		}
		if cfg.OpenVikingTimeout <= 0 {
			return fmt.Errorf("FRACTURING_SPACE_AI_OPENVIKING_TIMEOUT must be positive")
		}
		if cfg.OpenVikingMaxResults <= 0 {
			return fmt.Errorf("FRACTURING_SPACE_AI_OPENVIKING_MAX_RESULTS must be positive")
		}
		if cfg.OpenVikingMaxSections <= 0 {
			return fmt.Errorf("FRACTURING_SPACE_AI_OPENVIKING_MAX_SECTIONS must be positive")
		}
		if cfg.OpenVikingResourceSync <= 0 {
			return fmt.Errorf("FRACTURING_SPACE_AI_OPENVIKING_RESOURCE_SYNC_TIMEOUT must be positive")
		}
	}
	return nil
}

func (cfg runtimeConfig) encryptionKeyBytes() ([]byte, error) {
	encryptionKey := strings.TrimSpace(cfg.EncryptionKey)
	if encryptionKey == "" {
		return nil, fmt.Errorf("FRACTURING_SPACE_AI_ENCRYPTION_KEY is required")
	}
	keyBytes, err := decodeBase64Key(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	return keyBytes, nil
}

func (cfg runtimeConfig) buildSealer() (secret.Sealer, error) {
	keyBytes, err := cfg.encryptionKeyBytes()
	if err != nil {
		return nil, err
	}
	sealer, err := secret.NewAESGCMSealer(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("build secret sealer: %w", err)
	}
	return sealer, nil
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
		DBPath:                               strings.TrimSpace(srvEnv.DBPath),
		EncryptionKey:                        strings.TrimSpace(srvEnv.EncryptionKey),
		GameAddr:                             strings.TrimSpace(srvEnv.GameAddr),
		InternalServiceAllowlist:             parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
		OpenAIOAuthConfig:                    openAIOAuthConfig,
		OpenAIResponsesURL:                   strings.TrimSpace(srvEnv.OpenAIResponsesURL),
		AnthropicBaseURL:                     strings.TrimSpace(srvEnv.AnthropicBaseURL),
		OrchestrationTurnTimeout:             srvEnv.OrchestrationTurnTimeout,
		OrchestrationMaxSteps:                srvEnv.OrchestrationMaxSteps,
		ToolResultMaxBytes:                   srvEnv.ToolResultMaxBytes,
		DaggerheartReferenceRoot:             strings.TrimSpace(srvEnv.DaggerheartReferenceRoot),
		InstructionsRoot:                     strings.TrimSpace(srvEnv.InstructionsRoot),
		OpenVikingBaseURL:                    strings.TrimSpace(srvEnv.OpenVikingBaseURL),
		OpenVikingMode:                       strings.TrimSpace(srvEnv.OpenVikingMode),
		OpenVikingSessionSyncEnabled:         srvEnv.OpenVikingSessionSyncEnabled,
		OpenVikingAPIKey:                     strings.TrimSpace(srvEnv.OpenVikingAPIKey),
		OpenVikingTimeout:                    srvEnv.OpenVikingTimeout,
		OpenVikingMirrorRoot:                 strings.TrimSpace(srvEnv.OpenVikingMirrorRoot),
		OpenVikingVisibleMirrorRoot:          strings.TrimSpace(srvEnv.OpenVikingVisibleMirrorRoot),
		OpenVikingReferenceCorpusRoot:        strings.TrimSpace(srvEnv.OpenVikingReferenceCorpusRoot),
		OpenVikingReferenceCorpusVisibleRoot: strings.TrimSpace(srvEnv.OpenVikingReferenceCorpusVisibleRoot),
		OpenVikingMaxResults:                 srvEnv.OpenVikingMaxResults,
		OpenVikingMaxSections:                srvEnv.OpenVikingMaxSections,
		OpenVikingMinRelevanceScore:          srvEnv.OpenVikingMinRelevanceScore,
		OpenVikingResourceSync:               srvEnv.OpenVikingResourceSync,
		SessionGrantConfig:                   sessionGrantConfig,
	}
	if err := cfg.Validate(); err != nil {
		return runtimeConfig{}, err
	}
	return cfg, nil
}

func (cfg runtimeConfig) campaignTurnRunnerConfig(dialer orchestration.Dialer) orchestration.RunnerConfig {
	return orchestration.RunnerConfig{
		Dialer:             dialer,
		TurnPolicy:         orchestration.NewInteractionTurnPolicy(),
		MaxSteps:           cfg.OrchestrationMaxSteps,
		TurnTimeout:        cfg.OrchestrationTurnTimeout,
		ToolResultMaxBytes: cfg.ToolResultMaxBytes,
	}
}

// openAIOAuthConfig loads optional OpenAI OAuth config from the parsed server
// env. If all OpenAI OAuth variables are present they are wired in together;
// partial configuration is rejected to avoid accidental half-configured runtime.
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
