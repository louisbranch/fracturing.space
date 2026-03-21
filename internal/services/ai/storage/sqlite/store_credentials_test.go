package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutGetCredentialRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:abc",
		Status:           "active",
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
}

func TestListCredentialsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	for _, rec := range []storage.CredentialRecord{
		{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", SecretCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "cred-2", OwnerUserID: "user-1", Provider: "openai", Label: "B", SecretCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "cred-3", OwnerUserID: "user-2", Provider: "openai", Label: "C", SecretCiphertext: "enc:3", Status: "active", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutCredential(context.Background(), rec); err != nil {
			t.Fatalf("put credential %s: %v", rec.ID, err)
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

func TestPutCredentialUpsertPersistsRevocationLifecycle(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)
	revokedAt := now.Add(time.Minute)

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "A",
		SecretCiphertext: "enc:1",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "A",
		SecretCiphertext: "enc:1",
		Status:           "revoked",
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
	if got.Status != "revoked" {
		t.Fatalf("status = %q, want %q", got.Status, "revoked")
	}
	if got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("revoked_at = %v, want %v", got.RevokedAt, revokedAt)
	}
}

func TestPutCredentialRejectsDuplicateActiveLabelForOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 0, 0, 0, time.UTC)

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put first credential: %v", err)
	}

	err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-2",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            " primary ",
		SecretCiphertext: "enc:2",
		Status:           "active",
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

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put first credential: %v", err)
	}
	revokedAt := now.Add(time.Minute)
	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:1",
		Status:           "revoked",
		CreatedAt:        now,
		UpdatedAt:        revokedAt,
		RevokedAt:        &revokedAt,
	}); err != nil {
		t.Fatalf("put revoked credential: %v", err)
	}

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-2",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            " primary ",
		SecretCiphertext: "enc:2",
		Status:           "active",
		CreatedAt:        now.Add(2 * time.Minute),
		UpdatedAt:        now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("reuse revoked label: %v", err)
	}
}

func TestPutCredentialAllowsSameLabelAcrossOwners(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 10, 0, 0, time.UTC)

	for _, rec := range []storage.CredentialRecord{
		{
			ID:               "cred-1",
			OwnerUserID:      "user-1",
			Provider:         "openai",
			Label:            "Primary",
			SecretCiphertext: "enc:1",
			Status:           "active",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "cred-2",
			OwnerUserID:      "user-2",
			Provider:         "openai",
			Label:            " primary ",
			SecretCiphertext: "enc:2",
			Status:           "active",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	} {
		if err := store.PutCredential(context.Background(), rec); err != nil {
			t.Fatalf("put credential %s: %v", rec.ID, err)
		}
	}
}
