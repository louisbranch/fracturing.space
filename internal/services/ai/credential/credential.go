package credential

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
	// ProviderOpenAI is the only provider supported in phase 1.
	ProviderOpenAI Provider = "openai"
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
	// ErrInvalidProvider indicates unsupported provider value.
	ErrInvalidProvider = errors.New("provider is invalid")
	// ErrEmptyLabel indicates credential label is required.
	ErrEmptyLabel = errors.New("label is required")
	// ErrEmptySecret indicates secret input is required.
	ErrEmptySecret = errors.New("secret is required")
)

// Credential is the phase 1 domain model for a BYO provider credential.
type Credential struct {
	ID          string
	OwnerUserID string
	Provider    Provider
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
	Provider    Provider
	Label       string
	Secret      string
}

// NormalizeCreateInput trims and validates credential create input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}

	input.Provider = Provider(strings.ToLower(strings.TrimSpace(string(input.Provider))))
	if input.Provider != ProviderOpenAI {
		return CreateInput{}, ErrInvalidProvider
	}

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
