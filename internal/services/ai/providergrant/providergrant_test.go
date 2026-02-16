package providergrant

import (
	"testing"
	"time"
)

func TestNormalizeCreateInput(t *testing.T) {
	in := CreateInput{
		OwnerUserID:      " user-1 ",
		Provider:         Provider(" OPENAI "),
		GrantedScopes:    []string{" profile ", "", "profile", "responses.read"},
		TokenCiphertext:  " enc:abc ",
		RefreshSupported: true,
	}

	got, err := NormalizeCreateInput(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.OwnerUserID != "user-1" {
		t.Fatalf("owner_user_id = %q, want %q", got.OwnerUserID, "user-1")
	}
	if got.Provider != ProviderOpenAI {
		t.Fatalf("provider = %q, want %q", got.Provider, ProviderOpenAI)
	}
	if got.TokenCiphertext != "enc:abc" {
		t.Fatalf("token_ciphertext = %q, want %q", got.TokenCiphertext, "enc:abc")
	}
	if len(got.GrantedScopes) != 2 {
		t.Fatalf("granted_scopes len = %d, want %d", len(got.GrantedScopes), 2)
	}
	if got.GrantedScopes[0] != "profile" || got.GrantedScopes[1] != "responses.read" {
		t.Fatalf("granted_scopes = %v, want [profile responses.read]", got.GrantedScopes)
	}
}

func TestCreateProviderGrant(t *testing.T) {
	nowTime := time.Date(2026, 2, 15, 23, 20, 0, 0, time.UTC)
	now := func() time.Time { return nowTime }
	idGen := func() (string, error) { return "grant-1", nil }

	got, err := Create(CreateInput{
		OwnerUserID:     "user-1",
		Provider:        ProviderOpenAI,
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:token",
	}, now, idGen)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if got.ID != "grant-1" {
		t.Fatalf("id = %q, want %q", got.ID, "grant-1")
	}
	if got.Status != StatusActive {
		t.Fatalf("status = %q, want %q", got.Status, StatusActive)
	}
	if !got.CreatedAt.Equal(nowTime) || !got.UpdatedAt.Equal(nowTime) {
		t.Fatalf("timestamps = (%v, %v), want (%v, %v)", got.CreatedAt, got.UpdatedAt, nowTime, nowTime)
	}
}

func TestRevokeProviderGrant(t *testing.T) {
	createdAt := time.Date(2026, 2, 15, 23, 20, 0, 0, time.UTC)
	revokedAt := createdAt.Add(5 * time.Minute)
	grant := ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        ProviderOpenAI,
		TokenCiphertext: "enc:token",
		Status:          StatusActive,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}

	got, err := Revoke(grant, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if got.Status != StatusRevoked {
		t.Fatalf("status = %q, want %q", got.Status, StatusRevoked)
	}
	if got.RevokedAt == nil {
		t.Fatal("revoked_at is nil")
	}
	if !got.RevokedAt.Equal(revokedAt) || !got.UpdatedAt.Equal(revokedAt) {
		t.Fatalf("revocation timestamps = (%v, %v), want (%v, %v)", got.RevokedAt, got.UpdatedAt, revokedAt, revokedAt)
	}
}
