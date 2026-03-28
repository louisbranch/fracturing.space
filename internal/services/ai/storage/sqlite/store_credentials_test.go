package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetCredentialRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		SecretCiphertext: "enc:abc",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.PutCredential(context.Background(), input); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	got, err := store.GetCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.SecretCiphertext != input.SecretCiphertext {
		t.Fatalf("unexpected credential: %+v", got)
	}
	if got.Provider != provider.OpenAI {
		t.Fatalf("Provider = %q, want %q", got.Provider, provider.OpenAI)
	}
	if got.Status != credential.StatusActive {
		t.Fatalf("Status = %q, want %q", got.Status, credential.StatusActive)
	}
}

func TestListCredentialsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	for _, c := range []credential.Credential{
		{ID: "cred-1", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "A", SecretCiphertext: "enc:1", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "cred-2", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "B", SecretCiphertext: "enc:2", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "cred-3", OwnerUserID: "user-2", Provider: provider.OpenAI, Label: "C", SecretCiphertext: "enc:3", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutCredential(context.Background(), c); err != nil {
			t.Fatalf("put credential %s: %v", c.ID, err)
		}
	}

	page, err := store.ListCredentialsByOwner(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}
	if len(page.Credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(page.Credentials))
	}
}

func TestListCredentialsByOwnerPagination(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 50, 0, 0, time.UTC)

	for _, c := range []credential.Credential{
		{ID: "cred-1", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "A", SecretCiphertext: "enc:1", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "cred-2", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "B", SecretCiphertext: "enc:2", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "cred-3", OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "C", SecretCiphertext: "enc:3", Status: credential.StatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutCredential(context.Background(), c); err != nil {
			t.Fatalf("put credential %s: %v", c.ID, err)
		}
	}

	first, err := store.ListCredentialsByOwner(context.Background(), "user-1", 2, "")
	if err != nil {
		t.Fatalf("list first credential page: %v", err)
	}
	if len(first.Credentials) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.Credentials))
	}
	if first.NextPageToken != "cred-2" {
		t.Fatalf("first next page token = %q, want %q", first.NextPageToken, "cred-2")
	}

	second, err := store.ListCredentialsByOwner(context.Background(), "user-1", 2, first.NextPageToken)
	if err != nil {
		t.Fatalf("list second credential page: %v", err)
	}
	if len(second.Credentials) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.Credentials))
	}
	if second.Credentials[0].ID != "cred-3" {
		t.Fatalf("second page id = %q, want %q", second.Credentials[0].ID, "cred-3")
	}
	if second.NextPageToken != "" {
		t.Fatalf("second next page token = %q, want empty", second.NextPageToken)
	}
}

func TestPutCredentialUpsertPersistsRevocationLifecycle(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)
	revokedAt := now.Add(time.Minute)

	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "A",
		SecretCiphertext: "enc:1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "A",
		SecretCiphertext: "enc:1",
		Status:           credential.StatusRevoked,
		CreatedAt:        now,
		UpdatedAt:        revokedAt,
		RevokedAt:        &revokedAt,
	}); err != nil {
		t.Fatalf("put revoked credential: %v", err)
	}

	got, err := store.GetCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if got.Status != credential.StatusRevoked {
		t.Fatalf("status = %q, want %q", got.Status, credential.StatusRevoked)
	}
	if got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("revoked_at = %v, want %v", got.RevokedAt, revokedAt)
	}
}

func TestPutCredentialRejectsDuplicateActiveLabelForOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 0, 0, 0, time.UTC)

	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put first credential: %v", err)
	}

	err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-2",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            " primary ",
		SecretCiphertext: "enc:2",
		Status:           credential.StatusActive,
		CreatedAt:        now.Add(time.Minute),
		UpdatedAt:        now.Add(time.Minute),
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate active label error = %v, want storage.ErrConflict", err)
	}
}

func TestPutCredentialAllowsReuseAfterRevocation(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)

	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           credential.StatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put first credential: %v", err)
	}
	revokedAt := now.Add(time.Minute)
	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           credential.StatusRevoked,
		CreatedAt:        now,
		UpdatedAt:        revokedAt,
		RevokedAt:        &revokedAt,
	}); err != nil {
		t.Fatalf("put revoked credential: %v", err)
	}

	if err := store.PutCredential(context.Background(), credential.Credential{
		ID:               "cred-2",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            " primary ",
		SecretCiphertext: "enc:2",
		Status:           credential.StatusActive,
		CreatedAt:        now.Add(2 * time.Minute),
		UpdatedAt:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("reuse revoked label: %v", err)
	}
}

func TestPutCredentialAllowsSameLabelAcrossOwners(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 10, 0, 0, time.UTC)

	for _, c := range []credential.Credential{
		{
			ID:               "cred-1",
			OwnerUserID:      "user-1",
			Provider:         provider.OpenAI,
			Label:            "Primary",
			SecretCiphertext: "enc:1",
			Status:           credential.StatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "cred-2",
			OwnerUserID:      "user-2",
			Provider:         provider.OpenAI,
			Label:            " primary ",
			SecretCiphertext: "enc:2",
			Status:           credential.StatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	} {
		if err := store.PutCredential(context.Background(), c); err != nil {
			t.Fatalf("put credential %s: %v", c.ID, err)
		}
	}
}
