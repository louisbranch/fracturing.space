package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
)

func TestPutGetProviderGrantRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	input := providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read", "responses.write"},
		TokenCiphertext:  "enc:grant-token",
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        &expiresAt,
	}
	if err := store.PutProviderGrant(context.Background(), input); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	got, err := store.GetProviderGrant(context.Background(), "grant-1")
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.TokenCiphertext != input.TokenCiphertext {
		t.Fatalf("unexpected provider grant: %+v", got)
	}
	if got.Provider != provider.OpenAI {
		t.Fatalf("provider = %q, want %q", got.Provider, provider.OpenAI)
	}
	if got.Status != providergrant.StatusActive {
		t.Fatalf("status = %q, want %q", got.Status, providergrant.StatusActive)
	}
	if len(got.GrantedScopes) != 2 {
		t.Fatalf("granted_scopes len = %d, want 2", len(got.GrantedScopes))
	}
	if got.GrantedScopes[0] != "responses.read" || got.GrantedScopes[1] != "responses.write" {
		t.Fatalf("granted_scopes = %v, want [responses.read responses.write]", got.GrantedScopes)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestListProviderGrantsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)

	for _, grant := range []providergrant.ProviderGrant{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-2", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), grant); err != nil {
			t.Fatalf("put provider grant %s: %v", grant.ID, err)
		}
	}

	page, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", providergrant.Filter{})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(page.ProviderGrants) != 2 {
		t.Fatalf("expected 2 provider grants, got %d", len(page.ProviderGrants))
	}
}

func TestListProviderGrantsByOwnerPagination(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC)

	for _, grant := range []providergrant.ProviderGrant{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.manage"}, TokenCiphertext: "enc:3", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), grant); err != nil {
			t.Fatalf("put provider grant %s: %v", grant.ID, err)
		}
	}

	first, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 2, "", providergrant.Filter{})
	if err != nil {
		t.Fatalf("list first provider grant page: %v", err)
	}
	if len(first.ProviderGrants) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.ProviderGrants))
	}
	if first.NextPageToken != "grant-2" {
		t.Fatalf("first next page token = %q, want %q", first.NextPageToken, "grant-2")
	}

	second, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 2, first.NextPageToken, providergrant.Filter{})
	if err != nil {
		t.Fatalf("list second provider grant page: %v", err)
	}
	if len(second.ProviderGrants) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.ProviderGrants))
	}
	if second.ProviderGrants[0].ID != "grant-3" {
		t.Fatalf("second page id = %q, want %q", second.ProviderGrants[0].ID, "grant-3")
	}
	if second.NextPageToken != "" {
		t.Fatalf("second next page token = %q, want empty", second.NextPageToken)
	}
}

func TestListProviderGrantsByOwnerWithFilters(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 1, 0, 0, time.UTC)
	for _, grant := range []providergrant.ProviderGrant{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: "other", GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: providergrant.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-1", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: providergrant.StatusRevoked, CreatedAt: now, UpdatedAt: now},
		{ID: "grant-4", OwnerUserID: "user-2", Provider: provider.OpenAI, GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:4", Status: providergrant.StatusRevoked, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), grant); err != nil {
			t.Fatalf("put provider grant %s: %v", grant.ID, err)
		}
	}

	providerOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", providergrant.Filter{
		Provider: provider.OpenAI,
	})
	if err != nil {
		t.Fatalf("list provider-filtered grants: %v", err)
	}
	if len(providerOnly.ProviderGrants) != 2 {
		t.Fatalf("provider-filtered len = %d, want 2", len(providerOnly.ProviderGrants))
	}

	statusOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", providergrant.Filter{
		Status: providergrant.StatusRevoked,
	})
	if err != nil {
		t.Fatalf("list status-filtered grants: %v", err)
	}
	if len(statusOnly.ProviderGrants) != 1 {
		t.Fatalf("status-filtered len = %d, want 1", len(statusOnly.ProviderGrants))
	}
	if got := statusOnly.ProviderGrants[0].ID; got != "grant-3" {
		t.Fatalf("status-filtered id = %q, want %q", got, "grant-3")
	}
}

func TestPutProviderGrantUpsertPersistsLifecycleFields(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 50, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	if err := store.PutProviderGrant(context.Background(), providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  "enc:old",
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        &expiresAt,
	}); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	refreshedAt := now.Add(10 * time.Minute)
	newExpiresAt := refreshedAt.Add(2 * time.Hour)
	if err := store.PutProviderGrant(context.Background(), providergrant.ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  "enc:new",
		RefreshSupported: true,
		Status:           providergrant.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        refreshedAt,
		ExpiresAt:        &newExpiresAt,
		RefreshedAt:      &refreshedAt,
	}); err != nil {
		t.Fatalf("put updated provider grant: %v", err)
	}

	got, err := store.GetProviderGrant(context.Background(), "grant-1")
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if got.TokenCiphertext != "enc:new" {
		t.Fatalf("token_ciphertext = %q, want %q", got.TokenCiphertext, "enc:new")
	}
	if got.RefreshedAt == nil || !got.RefreshedAt.Equal(refreshedAt) {
		t.Fatalf("refreshed_at = %v, want %v", got.RefreshedAt, refreshedAt)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(newExpiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, newExpiresAt)
	}
}
