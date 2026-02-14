package passkey

import (
	"os"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

const (
	envRPID        = "FRACTURING_SPACE_WEBAUTHN_RP_ID"
	envRPName      = "FRACTURING_SPACE_WEBAUTHN_RP_DISPLAY_NAME"
	envRPOrigins   = "FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS"
	envSessionTTL  = "FRACTURING_SPACE_WEBAUTHN_SESSION_TTL"
	defaultRPID    = "localhost"
	defaultOrigin  = "http://localhost:8086"
	defaultSession = 5 * time.Minute
)

// SessionKind describes the WebAuthn session purpose.
type SessionKind string

const (
	SessionKindRegistration SessionKind = "registration"
	SessionKindLogin        SessionKind = "login"
)

// Config controls WebAuthn relying party settings.
type Config struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
	SessionTTL    time.Duration
}

// LoadConfigFromEnv returns passkey configuration with defaults.
func LoadConfigFromEnv() Config {
	rpID := strings.TrimSpace(os.Getenv(envRPID))
	if rpID == "" {
		rpID = defaultRPID
	}
	rpName := strings.TrimSpace(os.Getenv(envRPName))
	if rpName == "" {
		rpName = branding.AppName
	}

	origins := parseCSVEnv(envRPOrigins)
	if len(origins) == 0 {
		origins = []string{defaultOrigin}
	}

	sessionTTL := defaultSession
	if raw := strings.TrimSpace(os.Getenv(envSessionTTL)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			sessionTTL = parsed
		}
	}

	return Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     origins,
		SessionTTL:    sessionTTL,
	}
}

func parseCSVEnv(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
