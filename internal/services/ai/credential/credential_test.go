package credential

import (
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
