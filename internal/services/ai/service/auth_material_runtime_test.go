package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type fakeOAuthAdapter struct {
	buildAuthorizationURLErr error
	exchangeErr              error
	exchangeResult           provideroauth.TokenExchangeResult
	refreshErr               error
	refreshResult            provideroauth.TokenExchangeResult
	revokeErr                error

	lastAuthorizationInput provideroauth.AuthorizationURLInput
	lastRefreshToken       string
	lastRevokedToken       string
}

func (f *fakeOAuthAdapter) BuildAuthorizationURL(input provideroauth.AuthorizationURLInput) (string, error) {
	f.lastAuthorizationInput = input
	if f.buildAuthorizationURLErr != nil {
		return "", f.buildAuthorizationURLErr
	}
	return "https://provider.example.com/auth", nil
}

func (f *fakeOAuthAdapter) ExchangeAuthorizationCode(context.Context, provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	if f.exchangeErr != nil {
		return provideroauth.TokenExchangeResult{}, f.exchangeErr
	}
	if f.exchangeResult.TokenPayload.AccessToken == "" {
		return provideroauth.TokenExchangeResult{
			TokenPayload: provideroauth.TokenPayload{AccessToken: "at-1", RefreshToken: "rt-1"},
		}, nil
	}
	return f.exchangeResult, nil
}

func (f *fakeOAuthAdapter) RefreshToken(_ context.Context, input provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	f.lastRefreshToken = input.RefreshToken
	if f.refreshErr != nil {
		return provideroauth.TokenExchangeResult{}, f.refreshErr
	}
	return f.refreshResult, nil
}

func (f *fakeOAuthAdapter) RevokeToken(_ context.Context, input provideroauth.RevokeTokenInput) error {
	f.lastRevokedToken = input.Token
	return f.revokeErr
}

func TestProviderGrantRuntimeRefreshGrantSuccessUpdatesStoredToken(t *testing.T) {
	store := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	adapter := &fakeOAuthAdapter{
		refreshResult: provideroauth.TokenExchangeResult{
			TokenPayload: provideroauth.TokenPayload{AccessToken: "at-2", RefreshToken: "rt-2"},
			ExpiresAt:    ptrTime(now.Add(time.Hour)),
		},
	}
	runtime := NewProviderGrantRuntime(ProviderGrantRuntimeConfig{
		ProviderGrantStore: store,
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: adapter,
		}, nil, nil, nil),
		Sealer: &aifakes.Sealer{},
		Clock:  func() time.Time { return now },
	})

	got, err := runtime.RefreshGrant(context.Background(), "user-1", "grant-1")
	if err != nil {
		t.Fatalf("refresh provider grant: %v", err)
	}
	if adapter.lastRefreshToken != "rt-1" {
		t.Fatalf("refresh token = %q, want %q", adapter.lastRefreshToken, "rt-1")
	}
	if got.TokenCiphertext != `enc:{"access_token":"at-2","refresh_token":"rt-2"}` {
		t.Fatalf("token ciphertext = %q", got.TokenCiphertext)
	}
	if got.RefreshedAt == nil || !got.RefreshedAt.Equal(now) {
		t.Fatalf("refreshed_at = %v, want %v", got.RefreshedAt, now)
	}
}

func TestProviderGrantRuntimeRefreshGrantMarksRefreshFailed(t *testing.T) {
	store := aifakes.NewProviderGrantStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	adapter := &fakeOAuthAdapter{refreshErr: errors.New("provider timeout")}
	runtime := NewProviderGrantRuntime(ProviderGrantRuntimeConfig{
		ProviderGrantStore: store,
		ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
			provider.OpenAI: adapter,
		}, nil, nil, nil),
		Sealer: &aifakes.Sealer{},
		Clock:  func() time.Time { return now },
	})

	if _, err := runtime.RefreshGrant(context.Background(), "user-1", "grant-1"); err == nil {
		t.Fatal("expected refresh error")
	}
	updated := store.ProviderGrants["grant-1"]
	if updated.Status != "refresh_failed" {
		t.Fatalf("status = %q, want %q", updated.Status, "refresh_failed")
	}
	if updated.LastRefreshError == "" {
		t.Fatal("expected last_refresh_error to be set")
	}
}

func TestAuthMaterialResolverResolvesCredentialSecret(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "main",
		SecretCiphertext: "enc:sk-1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore: credentialStore,
		Sealer:          &aifakes.Sealer{},
	})

	token, err := resolver.ResolveAuthReferenceToken(context.Background(), "user-1", provider.OpenAI, agent.CredentialAuthReference("cred-1"))
	if err != nil {
		t.Fatalf("ResolveAuthReferenceToken: %v", err)
	}
	if token != "sk-1" {
		t.Fatalf("token = %q, want %q", token, "sk-1")
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
