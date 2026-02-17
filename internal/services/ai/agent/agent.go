// Package agent models user-owned AI runtime personalities.
//
// Agent definitions are intentionally metadata-first: handlers consume these records
// and decide credentials/grants at call time through dedicated resolution paths.
package agent

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

// Status represents lifecycle state for an AI agent.
type Status string

const (
	// StatusActive allows invocation and assignment usage.
	StatusActive Status = "active"
)

var (
	// ErrEmptyID indicates an ID is required.
	ErrEmptyID = errors.New("id is required")
	// ErrEmptyOwnerUserID indicates owner user ID is required.
	ErrEmptyOwnerUserID = errors.New("owner user id is required")
	// ErrEmptyName indicates agent name is required.
	ErrEmptyName = errors.New("name is required")
	// ErrInvalidProvider indicates unsupported provider value.
	ErrInvalidProvider = errors.New("provider is invalid")
	// ErrEmptyModel indicates model is required.
	ErrEmptyModel = errors.New("model is required")
	// ErrMissingAuthReference indicates one auth reference is required.
	ErrMissingAuthReference = errors.New("agent auth reference is required")
	// ErrMultipleAuthReferences indicates auth references are mutually exclusive.
	ErrMultipleAuthReferences = errors.New("exactly one agent auth reference is allowed")
)

// Agent is the phase 1 domain model for an AI profile configuration.
type Agent struct {
	ID              string
	OwnerUserID     string
	Name            string
	Provider        Provider
	Model           string
	CredentialID    string
	ProviderGrantID string
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateInput captures user-provided fields for creating an agent.
type CreateInput struct {
	OwnerUserID     string
	Name            string
	Provider        Provider
	Model           string
	CredentialID    string
	ProviderGrantID string
}

// UpdateInput captures mutable fields for updating an existing agent.
type UpdateInput struct {
	ID              string
	OwnerUserID     string
	Name            string
	Model           string
	CredentialID    string
	ProviderGrantID string
}

// NormalizeCreateInput validates and canonicalizes create input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateInput{}, ErrEmptyName
	}

	input.Provider = Provider(strings.ToLower(strings.TrimSpace(string(input.Provider))))
	if input.Provider != ProviderOpenAI {
		return CreateInput{}, ErrInvalidProvider
	}

	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" {
		return CreateInput{}, ErrEmptyModel
	}

	credentialID, providerGrantID, err := normalizeAuthReference(input.CredentialID, input.ProviderGrantID, true)
	if err != nil {
		return CreateInput{}, err
	}
	input.CredentialID = credentialID
	input.ProviderGrantID = providerGrantID

	return input, nil
}

// NormalizeUpdateInput validates and canonicalizes update input.
func NormalizeUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return UpdateInput{}, ErrEmptyID
	}

	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return UpdateInput{}, ErrEmptyOwnerUserID
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Model = strings.TrimSpace(input.Model)
	credentialID, providerGrantID, err := normalizeAuthReference(input.CredentialID, input.ProviderGrantID, false)
	if err != nil {
		return UpdateInput{}, err
	}
	input.CredentialID = credentialID
	input.ProviderGrantID = providerGrantID

	return input, nil
}

// Create constructs a normalized active agent with generated identifiers.
func Create(input CreateInput, now func() time.Time, idGenerator func() (string, error)) (Agent, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateInput(input)
	if err != nil {
		return Agent{}, err
	}

	agentID, err := idGenerator()
	if err != nil {
		return Agent{}, fmt.Errorf("generate agent id: %w", err)
	}

	createdAt := now().UTC()
	return Agent{
		ID:              agentID,
		OwnerUserID:     normalized.OwnerUserID,
		Name:            normalized.Name,
		Provider:        normalized.Provider,
		Model:           normalized.Model,
		CredentialID:    normalized.CredentialID,
		ProviderGrantID: normalized.ProviderGrantID,
		Status:          StatusActive,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}, nil
}

// normalizeAuthReference keeps agent auth references mutually exclusive to avoid
// ambiguous runtime secret-resolution behavior.
func normalizeAuthReference(credentialID string, providerGrantID string, require bool) (string, string, error) {
	credentialID = strings.TrimSpace(credentialID)
	providerGrantID = strings.TrimSpace(providerGrantID)

	hasCredential := credentialID != ""
	hasProviderGrant := providerGrantID != ""
	if hasCredential && hasProviderGrant {
		return "", "", ErrMultipleAuthReferences
	}
	if !hasCredential && !hasProviderGrant {
		if require {
			return "", "", ErrMissingAuthReference
		}
		return "", "", nil
	}
	return credentialID, providerGrantID, nil
}
