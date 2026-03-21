// Package agent models user-owned AI runtime personalities.
//
// Agent definitions are intentionally metadata-first: handlers consume these records
// and decide credentials/grants at call time through dedicated resolution paths.
package agent

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
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
	// ErrEmptyLabel indicates agent label is required.
	ErrEmptyLabel = errors.New("label is required")
	// ErrEmptyModel indicates model is required.
	ErrEmptyModel = errors.New("model is required")
	// ErrMissingAuthReference indicates one auth reference is required.
	ErrMissingAuthReference = errors.New("agent auth reference is required")
	// ErrMultipleAuthReferences indicates auth references are mutually exclusive.
	ErrMultipleAuthReferences = errors.New("exactly one agent auth reference is allowed")
	// ErrInvalidAuthReference indicates one typed auth reference is malformed.
	ErrInvalidAuthReference = errors.New("agent auth reference is invalid")
	// ErrInvalidLabel indicates agent label failed validation rules.
	ErrInvalidLabel = errors.New("agent label is invalid")
)

var labelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

// Agent is the phase 1 domain model for an AI profile configuration.
type Agent struct {
	ID            string
	OwnerUserID   string
	Label         string
	Instructions  string
	Provider      provider.Provider
	Model         string
	AuthReference AuthReference
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateInput captures user-provided fields for creating an agent.
type CreateInput struct {
	OwnerUserID   string
	Label         string
	Instructions  string
	Provider      provider.Provider
	Model         string
	AuthReference AuthReference
}

// UpdateInput captures mutable fields for updating an existing agent.
type UpdateInput struct {
	ID            string
	OwnerUserID   string
	Label         string
	Instructions  string
	Model         string
	AuthReference AuthReference
}

// NormalizeCreateInput validates and canonicalizes create input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}

	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" {
		return CreateInput{}, ErrEmptyLabel
	}
	if err := validateLabel(input.Label); err != nil {
		return CreateInput{}, err
	}
	input.Instructions = strings.TrimSpace(input.Instructions)

	normalizedProvider, err := provider.Normalize(string(input.Provider))
	if err != nil {
		return CreateInput{}, err
	}
	input.Provider = normalizedProvider

	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" {
		return CreateInput{}, ErrEmptyModel
	}

	authReference, err := NormalizeAuthReference(input.AuthReference, true)
	if err != nil {
		return CreateInput{}, err
	}
	input.AuthReference = authReference

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

	input.Label = strings.TrimSpace(input.Label)
	if input.Label != "" {
		if err := validateLabel(input.Label); err != nil {
			return UpdateInput{}, err
		}
	}
	input.Instructions = strings.TrimSpace(input.Instructions)
	input.Model = strings.TrimSpace(input.Model)
	authReference, err := NormalizeAuthReference(input.AuthReference, false)
	if err != nil {
		return UpdateInput{}, err
	}
	input.AuthReference = authReference

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
		ID:            agentID,
		OwnerUserID:   normalized.OwnerUserID,
		Label:         normalized.Label,
		Instructions:  normalized.Instructions,
		Provider:      normalized.Provider,
		Model:         normalized.Model,
		AuthReference: normalized.AuthReference,
		Status:        StatusActive,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}, nil
}

// ParseStatus trims and normalizes one persisted agent status.
func ParseStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(StatusActive):
		return StatusActive
	default:
		return ""
	}
}

// IsActive reports whether the agent status allows invocation and campaign use.
func (s Status) IsActive() bool {
	return ParseStatus(string(s)) == StatusActive
}

// Page is a paginated set of agents.
type Page struct {
	Agents        []Agent
	NextPageToken string
}

// AuthRefType returns which auth reference shape the agent uses.
func (a Agent) AuthRefType() string {
	return a.AuthReference.Type()
}

func validateLabel(value string) error {
	if !labelPattern.MatchString(value) {
		return ErrInvalidLabel
	}
	return nil
}
