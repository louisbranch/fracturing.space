package credential

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
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

func TestCredentialSecretCiphertextCarriedThrough(t *testing.T) {
	createdAt := time.Date(2026, 3, 18, 21, 0, 0, 0, time.UTC)
	cred := Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "primary",
		SecretCiphertext: "sealed",
		Status:           StatusActive,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}

	if cred.SecretCiphertext != "sealed" {
		t.Fatalf("SecretCiphertext = %q, want %q", cred.SecretCiphertext, "sealed")
	}
	if cred.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", cred.Status, StatusActive)
	}

	revokedAt := createdAt.Add(5 * time.Minute)
	revoked, err := Revoke(cred, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("Revoke error = %v", err)
	}
	if revoked.Status != StatusRevoked {
		t.Fatalf("Status = %q, want %q", revoked.Status, StatusRevoked)
	}
	// Ciphertext preserved through revocation.
	if revoked.SecretCiphertext != "sealed" {
		t.Fatalf("SecretCiphertext = %q, want %q", revoked.SecretCiphertext, "sealed")
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
