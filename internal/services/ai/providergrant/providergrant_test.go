package providergrant

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

func TestNormalizeCreateInput(t *testing.T) {
	in := CreateInput{
		OwnerUserID:      " user-1 ",
		Provider:         provider.Provider(" OPENAI "),
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
	if got.Provider != provider.OpenAI {
		t.Fatalf("provider = %q, want %q", got.Provider, provider.OpenAI)
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

func TestNormalizeCreateInputRejectsInvalidFields(t *testing.T) {
	_, err := NormalizeCreateInput(CreateInput{
		Provider:        provider.OpenAI,
		TokenCiphertext: "enc:abc",
	})
	if !errors.Is(err, ErrEmptyOwnerUserID) {
		t.Fatalf("NormalizeCreateInput() error = %v, want %v", err, ErrEmptyOwnerUserID)
	}

	_, err = NormalizeCreateInput(CreateInput{
		OwnerUserID:     "user-1",
		Provider:        provider.Provider("invalid"),
		TokenCiphertext: "enc:abc",
	})
	if err == nil {
		t.Fatal("expected provider normalization error")
	}

	_, err = NormalizeCreateInput(CreateInput{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
	})
	if !errors.Is(err, ErrEmptyTokenCiphertext) {
		t.Fatalf("NormalizeCreateInput() error = %v, want %v", err, ErrEmptyTokenCiphertext)
	}
}

func TestCreateProviderGrant(t *testing.T) {
	nowTime := time.Date(2026, 2, 15, 23, 20, 0, 0, time.UTC)
	now := func() time.Time { return nowTime }
	idGen := func() (string, error) { return "grant-1", nil }

	got, err := Create(CreateInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
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

func TestCreateProviderGrantReturnsIDGenerationError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := Create(CreateInput{
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: "enc:token",
	}, func() time.Time { return time.Date(2026, 2, 15, 23, 20, 0, 0, time.UTC) }, func() (string, error) {
		return "", wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestRevokeProviderGrant(t *testing.T) {
	createdAt := time.Date(2026, 2, 15, 23, 20, 0, 0, time.UTC)
	revokedAt := createdAt.Add(5 * time.Minute)
	grant := ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
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

func TestRevokeProviderGrantRejectsMissingFields(t *testing.T) {
	_, err := Revoke(ProviderGrant{OwnerUserID: "user-1"}, nil)
	if !errors.Is(err, ErrEmptyID) {
		t.Fatalf("Revoke() error = %v, want %v", err, ErrEmptyID)
	}

	_, err = Revoke(ProviderGrant{ID: "grant-1"}, nil)
	if !errors.Is(err, ErrEmptyOwnerUserID) {
		t.Fatalf("Revoke() error = %v, want %v", err, ErrEmptyOwnerUserID)
	}
}

func TestStatusAndGrantHelpers(t *testing.T) {
	expiresAt := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	grant := ProviderGrant{
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		RefreshSupported: true,
		Status:           Status(" active "),
		ExpiresAt:        &expiresAt,
	}

	if !grant.Status.IsActive() {
		t.Fatal("expected active status helper to normalize whitespace")
	}
	if !Status(" revoked ").IsRevoked() {
		t.Fatal("expected revoked status helper to normalize whitespace")
	}
	if grant.IsExpired(expiresAt.Add(-time.Second)) {
		t.Fatal("expected grant to be valid before expiry")
	}
	if !grant.IsExpired(expiresAt) {
		t.Fatal("expected grant to expire at expiry time")
	}
	if !grant.ShouldRefresh(expiresAt.Add(-time.Minute), 2*time.Minute) {
		t.Fatal("expected grant to refresh inside the refresh window")
	}
	if !grant.IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected grant to be usable by matching owner/provider")
	}
	if !grant.IsUsableBy("user-1", "") {
		t.Fatal("expected empty provider filter to allow the active grant")
	}
	if grant.IsUsableBy("user-2", provider.OpenAI) {
		t.Fatal("expected grant usability to reject wrong owner")
	}
	if (ProviderGrant{OwnerUserID: "user-1", Provider: provider.Provider(" invalid "), Status: StatusActive}).IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected invalid grant provider to reject usability")
	}
	if (ProviderGrant{OwnerUserID: "user-1", Provider: provider.OpenAI, Status: StatusRevoked}).IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected revoked grant to be unusable")
	}
	if (ProviderGrant{OwnerUserID: "user-1", Provider: provider.OpenAI, ExpiresAt: &expiresAt}).ShouldRefresh(expiresAt.Add(-time.Minute), time.Minute/2) {
		t.Fatal("expected grant without refresh support to skip refresh")
	}
	if got := ParseStatus(" refresh_failed "); got != StatusRefreshFailed {
		t.Fatalf("ParseStatus(refresh_failed) = %q, want %q", got, StatusRefreshFailed)
	}
	if got := ParseStatus(" expired "); got != StatusExpired {
		t.Fatalf("ParseStatus(expired) = %q, want %q", got, StatusExpired)
	}
	if got := ParseStatus("unknown"); got != "" {
		t.Fatalf("ParseStatus(unknown) = %q, want empty", got)
	}
}

func TestRecordRefreshSuccess(t *testing.T) {
	expiresAt := time.Date(2026, 3, 19, 1, 0, 0, 0, time.UTC)
	refreshedAt := time.Date(2026, 3, 18, 23, 10, 0, 0, time.UTC)
	got, err := RecordRefreshSuccess(ProviderGrant{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		TokenCiphertext:  "enc:old",
		RefreshSupported: true,
		Status:           StatusRefreshFailed,
		LastRefreshError: "old error",
	}, "enc:new", &expiresAt, refreshedAt)
	if err != nil {
		t.Fatalf("RecordRefreshSuccess error = %v", err)
	}
	if got.Status != StatusActive {
		t.Fatalf("status = %q, want %q", got.Status, StatusActive)
	}
	if got.TokenCiphertext != "enc:new" {
		t.Fatalf("token_ciphertext = %q, want %q", got.TokenCiphertext, "enc:new")
	}
	if got.LastRefreshError != "" {
		t.Fatalf("last_refresh_error = %q, want empty", got.LastRefreshError)
	}
	if got.RefreshedAt == nil || !got.RefreshedAt.Equal(refreshedAt) {
		t.Fatalf("refreshed_at = %v, want %v", got.RefreshedAt, refreshedAt)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestRecordRefreshSuccessRejectsInvalidFields(t *testing.T) {
	refreshedAt := time.Date(2026, 3, 18, 23, 10, 0, 0, time.UTC)

	_, err := RecordRefreshSuccess(ProviderGrant{OwnerUserID: "user-1"}, "enc:new", nil, refreshedAt)
	if !errors.Is(err, ErrEmptyID) {
		t.Fatalf("RecordRefreshSuccess() error = %v, want %v", err, ErrEmptyID)
	}

	_, err = RecordRefreshSuccess(ProviderGrant{ID: "grant-1"}, "enc:new", nil, refreshedAt)
	if !errors.Is(err, ErrEmptyOwnerUserID) {
		t.Fatalf("RecordRefreshSuccess() error = %v, want %v", err, ErrEmptyOwnerUserID)
	}

	_, err = RecordRefreshSuccess(ProviderGrant{ID: "grant-1", OwnerUserID: "user-1"}, " ", nil, refreshedAt)
	if !errors.Is(err, ErrEmptyTokenCiphertext) {
		t.Fatalf("RecordRefreshSuccess() error = %v, want %v", err, ErrEmptyTokenCiphertext)
	}
}

func TestRecordRefreshFailure(t *testing.T) {
	refreshedAt := time.Date(2026, 3, 18, 23, 12, 0, 0, time.UTC)
	got, err := RecordRefreshFailure(ProviderGrant{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        provider.OpenAI,
		TokenCiphertext: "enc:old",
		Status:          StatusActive,
	}, " provider error ", refreshedAt)
	if err != nil {
		t.Fatalf("RecordRefreshFailure error = %v", err)
	}
	if got.Status != StatusRefreshFailed {
		t.Fatalf("status = %q, want %q", got.Status, StatusRefreshFailed)
	}
	if got.LastRefreshError != "provider error" {
		t.Fatalf("last_refresh_error = %q, want %q", got.LastRefreshError, "provider error")
	}
	if got.RefreshedAt == nil || !got.RefreshedAt.Equal(refreshedAt) {
		t.Fatalf("refreshed_at = %v, want %v", got.RefreshedAt, refreshedAt)
	}
}

func TestRecordRefreshFailureRejectsInvalidFields(t *testing.T) {
	refreshedAt := time.Date(2026, 3, 18, 23, 12, 0, 0, time.UTC)

	_, err := RecordRefreshFailure(ProviderGrant{OwnerUserID: "user-1"}, "provider error", refreshedAt)
	if !errors.Is(err, ErrEmptyID) {
		t.Fatalf("RecordRefreshFailure() error = %v, want %v", err, ErrEmptyID)
	}

	_, err = RecordRefreshFailure(ProviderGrant{ID: "grant-1"}, "provider error", refreshedAt)
	if !errors.Is(err, ErrEmptyOwnerUserID) {
		t.Fatalf("RecordRefreshFailure() error = %v, want %v", err, ErrEmptyOwnerUserID)
	}

	_, err = RecordRefreshFailure(ProviderGrant{ID: "grant-1", OwnerUserID: "user-1"}, " ", refreshedAt)
	if !errors.Is(err, ErrEmptyRefreshError) {
		t.Fatalf("RecordRefreshFailure() error = %v, want %v", err, ErrEmptyRefreshError)
	}
}
