// Package providergrant models OAuth-connected provider access for AI runtime calls.
//
// Grants represent delegated authorization from an owner that can be refreshed and
// revoked without changing local credential ownership models.
package providergrant

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// Status represents provider grant lifecycle state.
type Status string

const (
	// StatusActive allows the grant to be used for provider calls.
	StatusActive Status = "active"
	// StatusRevoked blocks the grant from further usage.
	StatusRevoked Status = "revoked"
	// StatusExpired indicates token expiry with no usable refresh path.
	StatusExpired Status = "expired"
	// StatusRefreshFailed indicates the latest refresh attempt failed.
	StatusRefreshFailed Status = "refresh_failed"
)

var (
	// ErrEmptyOwnerUserID indicates owner user ID is required.
	ErrEmptyOwnerUserID = errors.New("owner user id is required")
	// ErrEmptyTokenCiphertext indicates sealed token payload is required.
	ErrEmptyTokenCiphertext = errors.New("token ciphertext is required")
	// ErrEmptyID indicates grant ID is required.
	ErrEmptyID = errors.New("id is required")
	// ErrEmptyRefreshError indicates refresh-failure detail is required.
	ErrEmptyRefreshError = errors.New("refresh error is required")
)

// ProviderGrant stores provider OAuth grant metadata.
type ProviderGrant struct {
	ID string

	OwnerUserID string
	Provider    provider.Provider

	GrantedScopes []string

	// TokenCiphertext contains encrypted provider token payload.
	TokenCiphertext string

	RefreshSupported bool
	Status           Status
	LastRefreshError string

	CreatedAt   time.Time
	UpdatedAt   time.Time
	RevokedAt   *time.Time
	ExpiresAt   *time.Time
	RefreshedAt *time.Time
}

// CreateInput contains fields required to create a provider grant.
type CreateInput struct {
	OwnerUserID string
	Provider    provider.Provider

	GrantedScopes []string

	TokenCiphertext  string
	RefreshSupported bool
	ExpiresAt        *time.Time
}

// NormalizeCreateInput trims and validates provider grant input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}

	normalizedProvider, err := provider.Normalize(string(input.Provider))
	if err != nil {
		return CreateInput{}, err
	}
	input.Provider = normalizedProvider

	input.TokenCiphertext = strings.TrimSpace(input.TokenCiphertext)
	if input.TokenCiphertext == "" {
		return CreateInput{}, ErrEmptyTokenCiphertext
	}

	input.GrantedScopes = NormalizeScopes(input.GrantedScopes)
	return input, nil
}

// NormalizeScopes deduplicates and trims a scope list, preserving insertion order.
func NormalizeScopes(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		scope := strings.TrimSpace(value)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Create constructs a normalized active provider grant with generated ID.
func Create(input CreateInput, now func() time.Time, idGenerator func() (string, error)) (ProviderGrant, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateInput(input)
	if err != nil {
		return ProviderGrant{}, err
	}

	grantID, err := idGenerator()
	if err != nil {
		return ProviderGrant{}, fmt.Errorf("generate provider grant id: %w", err)
	}

	createdAt := now().UTC()
	return ProviderGrant{
		ID:               grantID,
		OwnerUserID:      normalized.OwnerUserID,
		Provider:         normalized.Provider,
		GrantedScopes:    normalized.GrantedScopes,
		TokenCiphertext:  normalized.TokenCiphertext,
		RefreshSupported: normalized.RefreshSupported,
		Status:           StatusActive,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
		ExpiresAt:        normalized.ExpiresAt,
	}, nil
}

// Revoke marks a provider grant as revoked.
func Revoke(grant ProviderGrant, now func() time.Time) (ProviderGrant, error) {
	if now == nil {
		now = time.Now
	}
	grant.ID = strings.TrimSpace(grant.ID)
	if grant.ID == "" {
		return ProviderGrant{}, ErrEmptyID
	}
	if strings.TrimSpace(grant.OwnerUserID) == "" {
		return ProviderGrant{}, ErrEmptyOwnerUserID
	}

	revokedAt := now().UTC()
	grant.Status = StatusRevoked
	grant.RevokedAt = &revokedAt
	grant.UpdatedAt = revokedAt
	return grant, nil
}

// RecordRefreshSuccess applies one successful token refresh result.
func RecordRefreshSuccess(grant ProviderGrant, tokenCiphertext string, expiresAt *time.Time, refreshedAt time.Time) (ProviderGrant, error) {
	grant.ID = strings.TrimSpace(grant.ID)
	if grant.ID == "" {
		return ProviderGrant{}, ErrEmptyID
	}
	grant.OwnerUserID = strings.TrimSpace(grant.OwnerUserID)
	if grant.OwnerUserID == "" {
		return ProviderGrant{}, ErrEmptyOwnerUserID
	}
	tokenCiphertext = strings.TrimSpace(tokenCiphertext)
	if tokenCiphertext == "" {
		return ProviderGrant{}, ErrEmptyTokenCiphertext
	}

	refreshedAt = refreshedAt.UTC()
	grant.TokenCiphertext = tokenCiphertext
	grant.Status = StatusActive
	grant.LastRefreshError = ""
	grant.UpdatedAt = refreshedAt
	grant.RefreshedAt = &refreshedAt
	grant.ExpiresAt = expiresAt
	return grant, nil
}

// RecordRefreshFailure marks the grant unusable until another refresh succeeds.
func RecordRefreshFailure(grant ProviderGrant, refreshError string, refreshedAt time.Time) (ProviderGrant, error) {
	grant.ID = strings.TrimSpace(grant.ID)
	if grant.ID == "" {
		return ProviderGrant{}, ErrEmptyID
	}
	grant.OwnerUserID = strings.TrimSpace(grant.OwnerUserID)
	if grant.OwnerUserID == "" {
		return ProviderGrant{}, ErrEmptyOwnerUserID
	}
	refreshError = strings.TrimSpace(refreshError)
	if refreshError == "" {
		return ProviderGrant{}, ErrEmptyRefreshError
	}

	refreshedAt = refreshedAt.UTC()
	grant.Status = StatusRefreshFailed
	grant.LastRefreshError = refreshError
	grant.UpdatedAt = refreshedAt
	grant.RefreshedAt = &refreshedAt
	return grant, nil
}

// ParseStatus trims and normalizes one persisted provider-grant status.
func ParseStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(StatusActive):
		return StatusActive
	case string(StatusRevoked):
		return StatusRevoked
	case string(StatusExpired):
		return StatusExpired
	case string(StatusRefreshFailed):
		return StatusRefreshFailed
	default:
		return ""
	}
}

// FromRecord reconstructs a domain provider grant from a storage record for
// lifecycle and usability checks.
func FromRecord(record storage.ProviderGrantRecord) ProviderGrant {
	normalizedProvider, _ := provider.Normalize(record.Provider)
	return ProviderGrant{
		ID:               record.ID,
		OwnerUserID:      record.OwnerUserID,
		Provider:         normalizedProvider,
		GrantedScopes:    record.GrantedScopes,
		TokenCiphertext:  record.TokenCiphertext,
		RefreshSupported: record.RefreshSupported,
		Status:           ParseStatus(record.Status),
		LastRefreshError: record.LastRefreshError,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
		RevokedAt:        record.RevokedAt,
		ExpiresAt:        record.ExpiresAt,
		RefreshedAt:      record.LastRefreshedAt,
	}
}

// ApplyLifecycle writes domain-owned lifecycle changes back into the persisted
// provider-grant record.
func ApplyLifecycle(record *storage.ProviderGrantRecord, value ProviderGrant) {
	record.TokenCiphertext = value.TokenCiphertext
	record.Status = string(value.Status)
	record.LastRefreshError = value.LastRefreshError
	record.UpdatedAt = value.UpdatedAt
	record.RevokedAt = value.RevokedAt
	record.ExpiresAt = value.ExpiresAt
	record.LastRefreshedAt = value.RefreshedAt
}

// IsActive reports whether the grant is ready for use.
func (s Status) IsActive() bool {
	return ParseStatus(string(s)) == StatusActive
}

// IsRevoked reports whether the grant is explicitly revoked.
func (s Status) IsRevoked() bool {
	return ParseStatus(string(s)) == StatusRevoked
}

// IsExpired reports whether the grant has passed its expiry time.
func (g ProviderGrant) IsExpired(now time.Time) bool {
	if g.ExpiresAt == nil {
		return false
	}
	return !g.ExpiresAt.After(now)
}

// ShouldRefresh reports whether the grant should refresh before a call based on
// expiry proximity and refresh support.
func (g ProviderGrant) ShouldRefresh(now time.Time, window time.Duration) bool {
	if !g.RefreshSupported || g.ExpiresAt == nil {
		return false
	}
	return !g.ExpiresAt.After(now.Add(window))
}

// IsUsableBy reports whether the grant is active, owned by the caller, and
// matches the requested provider when one is supplied.
func (g ProviderGrant) IsUsableBy(ownerUserID string, requestedProvider provider.Provider) bool {
	if strings.TrimSpace(g.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !g.Status.IsActive() {
		return false
	}
	grantProvider, err := provider.Normalize(string(g.Provider))
	if err != nil {
		return false
	}
	if requestedProvider == "" {
		return true
	}
	return grantProvider == requestedProvider
}
