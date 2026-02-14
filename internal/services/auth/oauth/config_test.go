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
	if config.Providers != nil {
		t.Fatal("expected Providers to be nil")
	}
}

func TestParseCSVEnv(t *testing.T) {
	t.Setenv("TEST_CSV", " a, ,b ,c ")
	got := parseCSVEnv("TEST_CSV")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCSVEnv() = %v, want %v", got, want)
	}

	t.Setenv("TEST_CSV", "")
	if got := parseCSVEnv("TEST_CSV"); got != nil {
		t.Fatalf("parseCSVEnv(empty) = %v, want nil", got)
	}
}

func TestParseCSVEnvWithDefault(t *testing.T) {
	fallback := []string{"one", "two"}
	t.Setenv("TEST_DEFAULT", "")
	if got := parseCSVEnvWithDefault("TEST_DEFAULT", fallback); !reflect.DeepEqual(got, fallback) {
		t.Fatalf("parseCSVEnvWithDefault() = %v, want %v", got, fallback)
	}
}

func TestLoadConfigFromEnvParsesClientsAndUsers(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", `[{"client_id":"cli","client_secret":"secret","redirect_uris":["https://example.com/callback"],"token_endpoint_auth_method":"client_secret_post"}]`)
	t.Setenv("FRACTURING_SPACE_OAUTH_USERS", `[{"username":"u","password":"p","display_name":"User"}]`)

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

func TestParseClientsEnvInvalidJSON(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", "not-json")
	if clients := parseClientsEnv(); clients != nil {
		t.Fatalf("parseClientsEnv() = %v, want nil", clients)
	}
}
