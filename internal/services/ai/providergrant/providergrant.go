package providergrant

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// Provider identifies an AI provider integration.
type Provider string

const (
	// ProviderOpenAI is the only provider supported in this phase.
	ProviderOpenAI Provider = "openai"
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
	// ErrInvalidProvider indicates unsupported provider value.
	ErrInvalidProvider = errors.New("provider is invalid")
	// ErrEmptyTokenCiphertext indicates sealed token payload is required.
	ErrEmptyTokenCiphertext = errors.New("token ciphertext is required")
	// ErrEmptyID indicates grant ID is required.
	ErrEmptyID = errors.New("id is required")
)

// ProviderGrant stores provider OAuth grant metadata.
type ProviderGrant struct {
	ID string

	OwnerUserID string
	Provider    Provider

	GrantedScopes []string

	// TokenCiphertext contains encrypted provider token payload.
	TokenCiphertext string

	RefreshSupported bool
	Status           Status

	CreatedAt   time.Time
	UpdatedAt   time.Time
	RevokedAt   *time.Time
	ExpiresAt   *time.Time
	RefreshedAt *time.Time
}

// CreateInput contains fields required to create a provider grant.
type CreateInput struct {
	OwnerUserID string
	Provider    Provider

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

	input.Provider = Provider(strings.ToLower(strings.TrimSpace(string(input.Provider))))
	if input.Provider != ProviderOpenAI {
		return CreateInput{}, ErrInvalidProvider
	}

	input.TokenCiphertext = strings.TrimSpace(input.TokenCiphertext)
	if input.TokenCiphertext == "" {
		return CreateInput{}, ErrEmptyTokenCiphertext
	}

	input.GrantedScopes = normalizeScopes(input.GrantedScopes)
	return input, nil
}

func normalizeScopes(in []string) []string {
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
