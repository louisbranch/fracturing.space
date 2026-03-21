package credential

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestRevokeCredential(t *testing.T) {
	createdAt := time.Date(2026, 3, 18, 21, 0, 0, 0, time.UTC)
	revokedAt := createdAt.Add(5 * time.Minute)

	got, err := Revoke(Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "primary",
		Secret:      "secret",
		Status:      StatusActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("Revoke error = %v", err)
	}
	if got.Status != StatusRevoked {
		t.Fatalf("status = %q, want %q", got.Status, StatusRevoked)
	}
	if got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("revoked_at = %v, want %v", got.RevokedAt, revokedAt)
	}
	if !got.UpdatedAt.Equal(revokedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, revokedAt)
	}
}

func TestCreateCredential(t *testing.T) {
	fixedTime := time.Date(2026, 3, 18, 21, 0, 0, 0, time.UTC)

	got, err := Create(CreateInput{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "primary",
		Secret:      "sk-test",
	}, func() time.Time { return fixedTime }, func() (string, error) { return "cred-1", nil })
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if got.ID != "cred-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "cred-1")
	}
	if got.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", got.Status, StatusActive)
	}
	if got.Secret != "sk-test" {
		t.Fatalf("Secret = %q, want %q", got.Secret, "sk-test")
	}
}

func TestCreateCredentialValidation(t *testing.T) {
	_, err := Create(CreateInput{Label: "x", Secret: "s"}, nil, nil)
	if !errors.Is(err, ErrEmptyOwnerUserID) {
		t.Fatalf("expected ErrEmptyOwnerUserID, got %v", err)
	}

	_, err = Create(CreateInput{OwnerUserID: "u", Provider: provider.OpenAI, Secret: "s"}, nil, nil)
	if !errors.Is(err, ErrEmptyLabel) {
		t.Fatalf("expected ErrEmptyLabel, got %v", err)
	}

	_, err = Create(CreateInput{OwnerUserID: "u", Provider: provider.OpenAI, Label: "x"}, nil, nil)
	if !errors.Is(err, ErrEmptySecret) {
		t.Fatalf("expected ErrEmptySecret, got %v", err)
	}
}

func TestParseStatus(t *testing.T) {
	if ParseStatus("active") != StatusActive {
		t.Fatal("expected active")
	}
	if ParseStatus(" REVOKED ") != StatusRevoked {
		t.Fatal("expected revoked")
	}
	if ParseStatus("unknown") != "" {
		t.Fatal("expected empty for unknown")
	}
}

func TestStatusHelpers(t *testing.T) {
	if !StatusActive.IsActive() {
		t.Fatal("expected IsActive for active status")
	}
	if StatusActive.IsRevoked() {
		t.Fatal("expected not IsRevoked for active status")
	}
	if !StatusRevoked.IsRevoked() {
		t.Fatal("expected IsRevoked for revoked status")
	}
}

func TestFromRecordAndApplyLifecycle(t *testing.T) {
	createdAt := time.Date(2026, 3, 18, 21, 0, 0, 0, time.UTC)
	record := storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "primary",
		SecretCiphertext: "sealed",
		Status:           "active",
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}

	cred := FromRecord(record)
	if cred.ID != "cred-1" {
		t.Fatalf("ID = %q, want %q", cred.ID, "cred-1")
	}
	if cred.Provider != provider.OpenAI {
		t.Fatalf("Provider = %q, want %q", cred.Provider, provider.OpenAI)
	}
	if cred.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", cred.Status, StatusActive)
	}

	revokedAt := createdAt.Add(5 * time.Minute)
	cred.Status = StatusRevoked
	cred.UpdatedAt = revokedAt
	cred.RevokedAt = &revokedAt

	ApplyLifecycle(&record, cred)
	if record.Status != "revoked" {
		t.Fatalf("record.Status = %q, want %q", record.Status, "revoked")
	}
	if record.RevokedAt == nil || !record.RevokedAt.Equal(revokedAt) {
		t.Fatalf("record.RevokedAt = %v, want %v", record.RevokedAt, revokedAt)
	}
}

func TestIsUsableBy(t *testing.T) {
	cred := Credential{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Status:      StatusActive,
	}

	if !cred.IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected usable by matching owner/provider")
	}
	if !cred.IsUsableBy("user-1", "") {
		t.Fatal("expected usable when provider is empty")
	}
	if cred.IsUsableBy("user-2", provider.OpenAI) {
		t.Fatal("expected not usable by wrong owner")
	}

	revoked := cred
	revoked.Status = StatusRevoked
	if revoked.IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected not usable when revoked")
	}
}
