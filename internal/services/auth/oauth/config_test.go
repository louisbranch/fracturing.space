package oauth

import (
	"testing"
	"time"
)

func clearOAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_OAUTH_ISSUER", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_RESOURCE_SECRET", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_LOGIN_UI_URL", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI", "")
}

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	clearOAuthEnv(t)

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
	if config.LoginUIURL != "" {
		t.Fatalf("LoginUIURL = %q, want empty", config.LoginUIURL)
	}
}

func TestLoadConfigFromEnvParsesClientsAndLoginUIURL(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", `[{"client_id":"cli","client_secret":"secret","redirect_uris":["https://example.com/callback"],"token_endpoint_auth_method":"client_secret_post"}]`)
	t.Setenv("FRACTURING_SPACE_OAUTH_LOGIN_UI_URL", "https://web.example.com/login")

	config := LoadConfigFromEnv()
	if len(config.Clients) != 1 {
		t.Fatalf("Clients len = %d, want 1", len(config.Clients))
	}
	if config.Clients[0].ID != "cli" {
		t.Fatalf("Client ID = %q, want %q", config.Clients[0].ID, "cli")
	}
	if config.LoginUIURL != "https://web.example.com/login" {
		t.Fatalf("LoginUIURL = %q, want %q", config.LoginUIURL, "https://web.example.com/login")
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

func TestFirstPartyClientRegistration(t *testing.T) {
	t.Run("prepends trusted first-party client when env vars set", func(t *testing.T) {
		clearOAuthEnv(t)
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID", "fracturing-space")
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI", "http://localhost:8080/auth/callback")

		config := LoadConfigFromEnv()
		if len(config.Clients) != 1 {
			t.Fatalf("Clients len = %d, want 1", len(config.Clients))
		}
		client := config.Clients[0]
		if client.ID != "fracturing-space" {
			t.Fatalf("Client ID = %q, want %q", client.ID, "fracturing-space")
		}
		if !client.Trusted {
			t.Fatal("expected first-party client to be Trusted")
		}
		if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "http://localhost:8080/auth/callback" {
			t.Fatalf("RedirectURIs = %v, want [http://localhost:8080/auth/callback]", client.RedirectURIs)
		}
		if client.TokenEndpointAuthMethod != "none" {
			t.Fatalf("TokenEndpointAuthMethod = %q, want %q", client.TokenEndpointAuthMethod, "none")
		}
	})

	t.Run("first-party client prepended before JSON clients", func(t *testing.T) {
		clearOAuthEnv(t)
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID", "fracturing-space")
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI", "http://localhost:8080/auth/callback")
		t.Setenv("FRACTURING_SPACE_OAUTH_CLIENTS", `[{"client_id":"third-party","redirect_uris":["http://example.com/cb"]}]`)

		config := LoadConfigFromEnv()
		if len(config.Clients) != 2 {
			t.Fatalf("Clients len = %d, want 2", len(config.Clients))
		}
		if config.Clients[0].ID != "fracturing-space" {
			t.Fatalf("first client ID = %q, want %q", config.Clients[0].ID, "fracturing-space")
		}
		if config.Clients[0].Trusted != true {
			t.Fatal("first-party client should be trusted")
		}
		if config.Clients[1].ID != "third-party" {
			t.Fatalf("second client ID = %q, want %q", config.Clients[1].ID, "third-party")
		}
		if config.Clients[1].Trusted != false {
			t.Fatal("JSON client should not be trusted")
		}
	})

	t.Run("skipped when client ID is empty", func(t *testing.T) {
		clearOAuthEnv(t)
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID", "")
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI", "http://localhost:8080/auth/callback")

		config := LoadConfigFromEnv()
		if config.Clients != nil {
			t.Fatalf("Clients = %v, want nil", config.Clients)
		}
	})

	t.Run("skipped when redirect URI is empty", func(t *testing.T) {
		clearOAuthEnv(t)
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_CLIENT_ID", "fracturing-space")
		t.Setenv("FRACTURING_SPACE_OAUTH_FIRST_PARTY_REDIRECT_URI", "")

		config := LoadConfigFromEnv()
		if config.Clients != nil {
			t.Fatalf("Clients = %v, want nil", config.Clients)
		}
	})
}
