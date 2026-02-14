package oauth

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config describes the OAuth server configuration.
type Config struct {
	Issuer                  string
	ResourceSecret          string
	Clients                 []Client
	BootstrapUsers          []BootstrapUser
	LoginRedirectAllowlist  []string
	LoginUIURL              string
	Providers               map[string]ProviderConfig
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
}

// BootstrapUser seeds a local credentialed user.
type BootstrapUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// ProviderConfig describes an external OAuth provider configuration.
type ProviderConfig struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

// oauthEnv holds raw env values for OAuth configuration.
type oauthEnv struct {
	Issuer                  string        `env:"FRACTURING_SPACE_OAUTH_ISSUER"`
	ResourceSecret          string        `env:"FRACTURING_SPACE_OAUTH_RESOURCE_SECRET"`
	ClientsJSON             string        `env:"FRACTURING_SPACE_OAUTH_CLIENTS"`
	UsersJSON               string        `env:"FRACTURING_SPACE_OAUTH_USERS"`
	LoginRedirects          []string      `env:"FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS"    envSeparator:","`
	LoginUIURL              string        `env:"FRACTURING_SPACE_OAUTH_LOGIN_UI_URL"`
	TokenTTL                time.Duration `env:"FRACTURING_SPACE_OAUTH_TOKEN_TTL"           envDefault:"1h"`
	AuthorizationCodeTTL    time.Duration `env:"FRACTURING_SPACE_OAUTH_CODE_TTL"            envDefault:"10m"`
	PendingAuthorizationTTL time.Duration `env:"FRACTURING_SPACE_OAUTH_PENDING_TTL"         envDefault:"15m"`
	GoogleClientID          string        `env:"FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID"`
	GoogleClientSecret      string        `env:"FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURI       string        `env:"FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI"`
	GoogleScopes            []string      `env:"FRACTURING_SPACE_OAUTH_GOOGLE_SCOPES"       envSeparator:","`
	GitHubClientID          string        `env:"FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID"`
	GitHubClientSecret      string        `env:"FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET"`
	GitHubRedirectURI       string        `env:"FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI"`
	GitHubScopes            []string      `env:"FRACTURING_SPACE_OAUTH_GITHUB_SCOPES"       envSeparator:","`
}

// LoadConfigFromEnv loads OAuth server configuration from environment variables.
func LoadConfigFromEnv() Config {
	var raw oauthEnv
	if err := env.Parse(&raw); err != nil {
		return Config{
			TokenTTL:                time.Hour,
			AuthorizationCodeTTL:    10 * time.Minute,
			PendingAuthorizationTTL: 15 * time.Minute,
		}
	}

	var clients []Client
	if raw.ClientsJSON != "" {
		if err := json.Unmarshal([]byte(raw.ClientsJSON), &clients); err != nil {
			clients = nil
		}
	}

	var users []BootstrapUser
	if raw.UsersJSON != "" {
		if err := json.Unmarshal([]byte(raw.UsersJSON), &users); err != nil {
			users = nil
		}
	}

	// Trim empty entries from CSV-split slices.
	loginRedirects := trimCSV(raw.LoginRedirects)

	providers := buildProviders(raw)

	return Config{
		Issuer:                  raw.Issuer,
		ResourceSecret:          raw.ResourceSecret,
		Clients:                 clients,
		BootstrapUsers:          users,
		LoginRedirectAllowlist:  loginRedirects,
		LoginUIURL:              raw.LoginUIURL,
		Providers:               providers,
		TokenTTL:                raw.TokenTTL,
		AuthorizationCodeTTL:    raw.AuthorizationCodeTTL,
		PendingAuthorizationTTL: raw.PendingAuthorizationTTL,
	}
}

// trimCSV removes empty entries from a string slice.
func trimCSV(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func buildProviders(raw oauthEnv) map[string]ProviderConfig {
	providers := make(map[string]ProviderConfig)
	if raw.GoogleClientID != "" && raw.GoogleClientSecret != "" && raw.GoogleRedirectURI != "" {
		scopes := trimCSV(raw.GoogleScopes)
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}
		providers["google"] = ProviderConfig{
			Name:         "Google",
			ClientID:     raw.GoogleClientID,
			ClientSecret: raw.GoogleClientSecret,
			RedirectURI:  raw.GoogleRedirectURI,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
			Scopes:       scopes,
		}
	}
	if raw.GitHubClientID != "" && raw.GitHubClientSecret != "" && raw.GitHubRedirectURI != "" {
		scopes := trimCSV(raw.GitHubScopes)
		if len(scopes) == 0 {
			scopes = []string{"read:user", "user:email"}
		}
		providers["github"] = ProviderConfig{
			Name:         "GitHub",
			ClientID:     raw.GitHubClientID,
			ClientSecret: raw.GitHubClientSecret,
			RedirectURI:  raw.GitHubRedirectURI,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       scopes,
		}
	}
	if len(providers) == 0 {
		return nil
	}
	return providers
}
