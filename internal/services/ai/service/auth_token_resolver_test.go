package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type fakeOAuthAdapter struct {
	refreshErr    error
	refreshResult provider.TokenExchangeResult

	lastRefreshToken string
}

func (f *fakeOAuthAdapter) BuildAuthorizationURL(provider.AuthorizationURLInput) (string, error) {
	return "", errors.New("not implemented")
}

func (f *fakeOAuthAdapter) ExchangeAuthorizationCode(context.Context, provider.AuthorizationCodeInput) (provider.TokenExchangeResult, error) {
	return provider.TokenExchangeResult{}, errors.New("not implemented")
}

func (f *fakeOAuthAdapter) RefreshToken(_ context.Context, input provider.RefreshTokenInput) (provider.TokenExchangeResult, error) {
	f.lastRefreshToken = input.RefreshToken
	if f.refreshErr != nil {
		return provider.TokenExchangeResult{}, f.refreshErr
	}
	return f.refreshResult, nil
}

func (f *fakeOAuthAdapter) RevokeToken(context.Context, provider.RevokeTokenInput) error {
	return errors.New("not implemented")
}

func TestRefreshProviderGrantSuccessUpdatesStoredToken(t *testing.T) {
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
		refreshResult: provider.TokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	resolver := NewAuthTokenResolver(AuthTokenResolverConfig{
		ProviderGrantStore: store,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: adapter,
		},
		Sealer: &aifakes.Sealer{},
		Clock:  func() time.Time { return now },
	})

	got, err := resolver.refreshProviderGrant(context.Background(), "user-1", "grant-1")
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

func TestRefreshProviderGrantMarksRefreshFailed(t *testing.T) {
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
	resolver := NewAuthTokenResolver(AuthTokenResolverConfig{
		ProviderGrantStore: store,
		ProviderOAuthAdapters: map[provider.Provider]provider.OAuthAdapter{
			provider.OpenAI: adapter,
		},
		Sealer: &aifakes.Sealer{},
		Clock:  func() time.Time { return now },
	})

	if _, err := resolver.refreshProviderGrant(context.Background(), "user-1", "grant-1"); err == nil {
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

func ptrTime(value time.Time) *time.Time {
	return &value
}
