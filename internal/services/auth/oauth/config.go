package oauth

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config describes first-party OAuth authorization-server behavior.
type Config struct {
	Issuer                  string
	ResourceSecret          string
	Clients                 []Client
	LoginUIURL              string
	TokenTTL                time.Duration
	AuthorizationCodeTTL    time.Duration
	PendingAuthorizationTTL time.Duration
}

// Client represents a registered OAuth client application.
type Client struct {
	ID                      string   `json:"client_id"`
	Secret                  string   `json:"client_secret,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	Name                    string   `json:"client_name,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	// Trusted marks first-party clients that skip the consent screen.
	Trusted bool `json:"-"`
}

// oauthEnv holds raw env values for OAuth configuration.
type oauthEnv struct {
	Issuer                  string        `env:"FRACTURING_SPACE_OAUTH_ISSUER"`
	ResourceSecret          string        `env:"FRACTURING_SPACE_OAUTH_RESOURCE_SECRET"`
	ClientsJSON             string        `env:"FRACTURING_SPACE_OAUTH_CLIENTS"`
	LoginUIURL              string        `env:"FRACTURING_SPACE_OAUTH_LOGIN_UI_URL"`
	TokenTTL                time.Duration `env:"FRACTURING_SPACE_OAUTH_TOKEN_TTL"           envDefault:"1h"`
	AuthorizationCodeTTL    time.Duration `env:"FRACTURING_SPACE_OAUTH_CODE_TTL"            envDefault:"10m"`
	PendingAuthorizationTTL time.Duration `env:"FRACTURING_SPACE_OAUTH_PENDING_TTL"         envDefault:"15m"`
	FirstPartyClientID      string        `env:"FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID"`
	FirstPartyRedirectURI   string        `env:"FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI"`
}

// LoadConfigFromEnv loads authorization-server configuration and applies safe defaults.
func LoadConfigFromEnv() Config {
	var raw oauthEnv
	_ = env.Parse(&raw)
	if raw.TokenTTL == 0 {
		raw.TokenTTL = time.Hour
	}
	if raw.AuthorizationCodeTTL == 0 {
		raw.AuthorizationCodeTTL = 10 * time.Minute
	}
	if raw.PendingAuthorizationTTL == 0 {
		raw.PendingAuthorizationTTL = 15 * time.Minute
	}

	var clients []Client
	if raw.ClientsJSON != "" {
		if err := json.Unmarshal([]byte(raw.ClientsJSON), &clients); err != nil {
			clients = nil
		}
	}

	// Prepend trusted first-party client when both ID and redirect URI are set.
	fpID := strings.TrimSpace(raw.FirstPartyClientID)
	fpRedirect := strings.TrimSpace(raw.FirstPartyRedirectURI)
	if fpID != "" && fpRedirect != "" {
		fp := Client{
			ID:                      fpID,
			RedirectURIs:            []string{fpRedirect},
			Name:                    "Fracturing Space",
			TokenEndpointAuthMethod: "none",
			Trusted:                 true,
		}
		clients = append([]Client{fp}, clients...)
	}

	return Config{
		Issuer:                  raw.Issuer,
		ResourceSecret:          raw.ResourceSecret,
		Clients:                 clients,
		LoginUIURL:              raw.LoginUIURL,
		TokenTTL:                raw.TokenTTL,
		AuthorizationCodeTTL:    raw.AuthorizationCodeTTL,
		PendingAuthorizationTTL: raw.PendingAuthorizationTTL,
	}
}
