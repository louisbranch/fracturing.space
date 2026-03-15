// Package credential models BYO LLM provider credentials for AI operations.
//
// Credentials are validated here as domain objects and encrypted by higher
// layers before persistence, so callers can reason in plain terms without
// leaking plaintext secrets.
package credential

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

// Status represents credential lifecycle state.
type Status string

const (
	// StatusActive allows credential usage.
	StatusActive Status = "active"
	// StatusRevoked blocks credential usage.
	StatusRevoked Status = "revoked"
)

var (
	// ErrEmptyOwnerUserID indicates owner user ID is required.
	ErrEmptyOwnerUserID = errors.New("owner user id is required")
	// ErrEmptyLabel indicates credential label is required.
	ErrEmptyLabel = errors.New("label is required")
	// ErrEmptySecret indicates secret input is required.
	ErrEmptySecret = errors.New("secret is required")
)

// Credential is the phase 1 domain model for a BYO provider credential.
type Credential struct {
	ID          string
	OwnerUserID string
	Provider    provider.Provider
	Label       string
	// Secret is plaintext in the domain model; the service/storage boundary is
	// responsible for encryption before persistence.
	Secret    string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
	RevokedAt *time.Time
}

// CreateInput contains user-provided fields needed to create a credential.
type CreateInput struct {
	OwnerUserID string
	Provider    provider.Provider
	Label       string
	Secret      string
}

// NormalizeCreateInput trims and validates credential create input.
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

	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" {
		return CreateInput{}, ErrEmptyLabel
	}

	input.Secret = strings.TrimSpace(input.Secret)
	if input.Secret == "" {
		return CreateInput{}, ErrEmptySecret
	}

	return input, nil
}

// Create constructs a normalized active credential with generated identifiers.
func Create(input CreateInput, now func() time.Time, idGenerator func() (string, error)) (Credential, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateInput(input)
	if err != nil {
		return Credential{}, err
	}

	credentialID, err := idGenerator()
	if err != nil {
		return Credential{}, fmt.Errorf("generate credential id: %w", err)
	}

	createdAt := now().UTC()
	return Credential{
		ID:          credentialID,
		OwnerUserID: normalized.OwnerUserID,
		Provider:    normalized.Provider,
		Label:       normalized.Label,
		Secret:      normalized.Secret,
		Status:      StatusActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

// ParseStatus trims and normalizes one persisted credential status.
func ParseStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(StatusActive):
		return StatusActive
	case string(StatusRevoked):
		return StatusRevoked
	default:
		return ""
	}
}

// IsActive reports whether the credential may still be used.
func (s Status) IsActive() bool {
	return ParseStatus(string(s)) == StatusActive
}

// IsRevoked reports whether the credential is explicitly revoked.
func (s Status) IsRevoked() bool {
	return ParseStatus(string(s)) == StatusRevoked
}

// IsUsableBy reports whether the credential is active, owned by the caller, and
// matches the requested provider when one is supplied.
func (c Credential) IsUsableBy(ownerUserID string, requestedProvider provider.Provider) bool {
	if strings.TrimSpace(c.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !c.Status.IsActive() {
		return false
	}
	credentialProvider, err := provider.Normalize(string(c.Provider))
	if err != nil {
		return false
	}
	if requestedProvider == "" {
		return true
	}
	return credentialProvider == requestedProvider
}
