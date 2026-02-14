package oauth

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_ISSUER", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_RESOURCE_SECRET", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_USERS", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_LOGIN_UI_URL", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI", "")

	config := LoadConfigFromEnv()
	if config.Issuer != "" {
		t.Fatalf("Issuer = %q, want empty", config.Issuer)
	}
	if config.ResourceSecret != "" {
		t.Fatalf("ResourceSecret = %q, want empty", config.ResourceSecret)
	}
	if config.TokenTTL != time.Hour {
		t.Fatalf("TokenTTL = %v, want %v", config.TokenTTL, time.Hour)
	}
	if config.AuthorizationCodeTTL != 10*time.Minute {
		t.Fatalf("AuthorizationCodeTTL = %v, want %v", config.AuthorizationCodeTTL, 10*time.Minute)
	}
	if config.PendingAuthorizationTTL != 15*time.Minute {
		t.Fatalf("PendingAuthorizationTTL = %v, want %v", config.PendingAuthorizationTTL, 15*time.Minute)
	}
	if config.Clients != nil {
		t.Fatal("expected Clients to be nil")
	}
	if config.BootstrapUsers != nil {
		t.Fatal("expected BootstrapUsers to be nil")
	}
	if config.LoginRedirectAllowlist != nil {
		t.Fatal("expected LoginRedirectAllowlist to be nil")
	}
	if config.LoginUIURL != "" {
		t.Fatalf("LoginUIURL = %q, want empty", config.LoginUIURL)
	}
	if config.Providers != nil {
		t.Fatal("expected Providers to be nil")
	}
}

func TestTrimCSV(t *testing.T) {
	got := trimCSV([]string{" a", " ", "b ", "c "})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("trimCSV() = %v, want %v", got, want)
	}

	if got := trimCSV(nil); got != nil {
		t.Fatalf("trimCSV(nil) = %v, want nil", got)
	}

	if got := trimCSV([]string{"", " "}); got != nil {
		t.Fatalf("trimCSV(empty) = %v, want nil", got)
	}
}

func TestLoadConfigFromEnvParsesClientsAndUsers(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", `[{"client_id":"cli","client_secret":"secret","redirect_uris":["https://example.com/callback"],"token_endpoint_auth_method":"client_secret_post"}]`)
	t.Setenv("FRACTURING_SPACE_OAUTH_USERS", `[{"username":"u","password":"p","display_name":"User"}]`)
	t.Setenv("FRACTURING_SPACE_OAUTH_LOGIN_UI_URL", "https://web.example.com/login")

	config := LoadConfigFromEnv()
	if len(config.Clients) != 1 {
		t.Fatalf("Clients len = %d, want 1", len(config.Clients))
	}
	if config.Clients[0].ID != "cli" {
		t.Fatalf("Client ID = %q, want %q", config.Clients[0].ID, "cli")
	}
	if len(config.BootstrapUsers) != 1 {
		t.Fatalf("BootstrapUsers len = %d, want 1", len(config.BootstrapUsers))
	}
	if config.BootstrapUsers[0].Username != "u" {
		t.Fatalf("BootstrapUsers[0].Username = %q, want %q", config.BootstrapUsers[0].Username, "u")
	}
	if config.LoginUIURL != "https://web.example.com/login" {
		t.Fatalf("LoginUIURL = %q, want %q", config.LoginUIURL, "https://web.example.com/login")
	}
}

func TestLoadConfigFromEnvProviders(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_ID", "gid")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_CLIENT_SECRET", "gsecret")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_REDIRECT_URI", "https://example.com/google")
	t.Setenv("FRACTURING_SPACE_OAUTH_GOOGLE_SCOPES", "openid,email")
	// Partial GitHub config should be ignored.
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_ID", "hid")
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_GITHUB_REDIRECT_URI", "https://example.com/github")

	config := LoadConfigFromEnv()
	if len(config.Providers) != 1 {
		t.Fatalf("Providers len = %d, want 1", len(config.Providers))
	}
	google, ok := config.Providers["google"]
	if !ok {
		t.Fatal("expected google provider")
	}
	if google.ClientID != "gid" {
		t.Fatalf("google.ClientID = %q, want %q", google.ClientID, "gid")
	}
	if !reflect.DeepEqual(google.Scopes, []string{"openid", "email"}) {
		t.Fatalf("google.Scopes = %v, want %v", google.Scopes, []string{"openid", "email"})
	}
}

func TestLoadConfigFromEnvInvalidClientsJSON(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", "not-json")
	config := LoadConfigFromEnv()
	if config.Clients != nil {
		t.Fatalf("Clients = %v, want nil", config.Clients)
	}
}

func TestLoadConfigFromEnvInvalidTokenTTLKeepsIssuer(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_ISSUER", "https://issuer.example.com")
	t.Setenv("FRACTURING_SPACE_OAUTH_TOKEN_TTL", "not-a-duration")

	config := LoadConfigFromEnv()
	if config.Issuer != "https://issuer.example.com" {
		t.Fatalf("Issuer = %q, want %q", config.Issuer, "https://issuer.example.com")
	}
	if config.TokenTTL != time.Hour {
		t.Fatalf("TokenTTL = %v, want %v", config.TokenTTL, time.Hour)
	}
}
