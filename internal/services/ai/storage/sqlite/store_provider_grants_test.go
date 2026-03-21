package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetProviderGrantRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	input := storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read", "responses.write"},
		TokenCiphertext:  "enc:grant-token",
		RefreshSupported: true,
		Status:           "active",
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

	for _, rec := range []storage.ProviderGrantRecord{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-2", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: "active", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), rec); err != nil {
			t.Fatalf("put provider grant %s: %v", rec.ID, err)
		}
	}

	page, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(page.ProviderGrants) != 2 {
		t.Fatalf("expected 2 provider grants, got %d", len(page.ProviderGrants))
	}
}

func TestListProviderGrantsByOwnerWithFilters(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 1, 0, 0, time.UTC)
	for _, rec := range []storage.ProviderGrantRecord{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: "other", GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: "revoked", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-4", OwnerUserID: "user-2", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:4", Status: "revoked", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), rec); err != nil {
			t.Fatalf("put provider grant %s: %v", rec.ID, err)
		}
	}

	providerOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{
		Provider: "openai",
	})
	if err != nil {
		t.Fatalf("list provider-filtered grants: %v", err)
	}
	if len(providerOnly.ProviderGrants) != 2 {
		t.Fatalf("provider-filtered len = %d, want 2", len(providerOnly.ProviderGrants))
	}

	statusOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{
		Status: "revoked",
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

	if err := store.PutProviderGrant(context.Background(), storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  "enc:old",
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        &expiresAt,
	}); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	refreshedAt := now.Add(10 * time.Minute)
	newExpiresAt := refreshedAt.Add(2 * time.Hour)
	if err := store.PutProviderGrant(context.Background(), storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  "enc:new",
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        refreshedAt,
		ExpiresAt:        &newExpiresAt,
		LastRefreshedAt:  &refreshedAt,
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
	if got.LastRefreshedAt == nil || !got.LastRefreshedAt.Equal(refreshedAt) {
		t.Fatalf("last_refreshed_at = %v, want %v", got.LastRefreshedAt, refreshedAt)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(newExpiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, newExpiresAt)
	}
}
