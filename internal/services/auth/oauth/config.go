package oauth

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

const (
	defaultTokenTTL   = time.Hour
	defaultCodeTTL    = 10 * time.Minute
	defaultPendingTTL = 15 * time.Minute
)

// Config describes the OAuth server configuration.
type Config struct {
	Issuer                  string
	ResourceSecret          string
	Clients                 []Client
	BootstrapUsers          []BootstrapUser
	LoginRedirectAllowlist  []string
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

// LoadConfigFromEnv loads OAuth server configuration from environment variables.
func LoadConfigFromEnv() Config {
	issuer := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_ISSUER"))
	resourceSecret := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_RESOURCE_SECRET"))
	clients := parseClientsEnv()
	users := parseBootstrapUsersEnv()
	loginRedirects := parseCSVEnv("FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS")
	providers := loadProvidersFromEnv()

	return Config{
		Issuer:                  issuer,
		ResourceSecret:          resourceSecret,
		Clients:                 clients,
		BootstrapUsers:          users,
		LoginRedirectAllowlist:  loginRedirects,
		Providers:               providers,
		TokenTTL:                defaultTokenTTL,
		AuthorizationCodeTTL:    defaultCodeTTL,
		PendingAuthorizationTTL: defaultPendingTTL,
	}
}

func parseClientsEnv() []Client {
	payload := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_CLIENTS"))
	if payload == "" {
		return nil
	}

	var clients []Client
	if err := json.Unmarshal([]byte(payload), &clients); err != nil {
		return nil
	}
	return clients
}

func parseBootstrapUsersEnv() []BootstrapUser {
	payload := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_USERS"))
	if payload == "" {
		return nil
	}

	var users []BootstrapUser
	if err := json.Unmarshal([]byte(payload), &users); err != nil {
		return nil
	}
	return users
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func loadProvidersFromEnv() map[string]ProviderConfig {
	providers := make(map[string]ProviderConfig)
	if google := loadGoogleProvider(); google.ClientID != "" && google.ClientSecret != "" && google.RedirectURI != "" {
		providers["google"] = google
	}
	if github := loadGitHubProvider(); github.ClientID != "" && github.ClientSecret != "" && github.RedirectURI != "" {
		providers["github"] = github
	}
	if len(providers) == 0 {
		return nil
	}
	return providers
}

func loadGoogleProvider() ProviderConfig {
	return ProviderConfig{
		Name:         "Google",
		ClientID:     strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET")),
		RedirectURI:  strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI")),
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       parseCSVEnvWithDefault("FRACTURING_SPACE_OAUTH_GOOGLE_SCOPES", []string{"openid", "email", "profile"}),
	}
}

func loadGitHubProvider() ProviderConfig {
	return ProviderConfig{
		Name:         "GitHub",
		ClientID:     strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET")),
		RedirectURI:  strings.TrimSpace(os.Getenv("FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI")),
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       parseCSVEnvWithDefault("FRACTURING_SPACE_OAUTH_GITHUB_SCOPES", []string{"read:user", "user:email"}),
	}
}

func parseCSVEnvWithDefault(key string, fallback []string) []string {
	values := parseCSVEnv(key)
	if len(values) == 0 {
		return fallback
	}
	return values
}
