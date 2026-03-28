// Package providerconnect models one provider OAuth connect-session lifecycle.
//
// Connect sessions are temporary handshake records used during PKCE-backed
// provider authorization. They are support workflow state, not long-lived AI
// runtime auth like provider grants.
package providerconnect

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

// Status represents provider connect-session lifecycle state.
type Status string

const (
	// StatusPending means the outbound OAuth handshake is still awaiting code
	// exchange completion.
	StatusPending Status = "pending"
	// StatusCompleted means the connect handshake completed successfully.
	StatusCompleted Status = "completed"
)

var (
	// ErrEmptyID indicates session ID is required.
	ErrEmptyID = errors.New("id is required")
	// ErrEmptyOwnerUserID indicates owner user ID is required.
	ErrEmptyOwnerUserID = errors.New("owner user id is required")
	// ErrEmptyStateHash indicates state hash is required.
	ErrEmptyStateHash = errors.New("state hash is required")
	// ErrEmptyCodeVerifierCiphertext indicates sealed verifier material is required.
	ErrEmptyCodeVerifierCiphertext = errors.New("code verifier ciphertext is required")
	// ErrEmptyExpiresAt indicates expiry is required.
	ErrEmptyExpiresAt = errors.New("expires at is required")
)

// Session stores one provider OAuth connect handshake.
type Session struct {
	ID              string
	OwnerUserID     string
	Provider        provider.Provider
	Status          Status
	RequestedScopes []string

	// StateHash stores a non-reversible hash of the outbound OAuth state token.
	StateHash string
	// CodeVerifierCiphertext stores encrypted PKCE verifier material.
	CodeVerifierCiphertext string

	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	CompletedAt *time.Time
}

// Store persists provider OAuth connect-session state.
type Store interface {
	PutProviderConnectSession(ctx context.Context, session Session) error
	GetProviderConnectSession(ctx context.Context, connectSessionID string) (Session, error)
	CompleteProviderConnectSession(ctx context.Context, session Session) error
}

// CreateInput contains fields required to start a connect session.
type CreateInput struct {
	ID                     string
	OwnerUserID            string
	Provider               provider.Provider
	RequestedScopes        []string
	StateHash              string
	CodeVerifierCiphertext string
	CreatedAt              time.Time
	ExpiresAt              time.Time
}

// NormalizeCreateInput trims and validates connect-session input.
func NormalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return CreateInput{}, ErrEmptyID
	}
	input.OwnerUserID = strings.TrimSpace(input.OwnerUserID)
	if input.OwnerUserID == "" {
		return CreateInput{}, ErrEmptyOwnerUserID
	}
	normalizedProvider, err := provider.Normalize(string(input.Provider))
	if err != nil {
		return CreateInput{}, err
	}
	input.Provider = normalizedProvider
	input.RequestedScopes = provideroauth.NormalizeScopes(input.RequestedScopes)
	input.StateHash = strings.TrimSpace(input.StateHash)
	if input.StateHash == "" {
		return CreateInput{}, ErrEmptyStateHash
	}
	input.CodeVerifierCiphertext = strings.TrimSpace(input.CodeVerifierCiphertext)
	if input.CodeVerifierCiphertext == "" {
		return CreateInput{}, ErrEmptyCodeVerifierCiphertext
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	} else {
		input.CreatedAt = input.CreatedAt.UTC()
	}
	if input.ExpiresAt.IsZero() {
		return CreateInput{}, ErrEmptyExpiresAt
	}
	input.ExpiresAt = input.ExpiresAt.UTC()
	return input, nil
}

// CreatePending constructs a normalized pending connect session.
func CreatePending(input CreateInput) (Session, error) {
	normalized, err := NormalizeCreateInput(input)
	if err != nil {
		return Session{}, err
	}
	return Session{
		ID:                     normalized.ID,
		OwnerUserID:            normalized.OwnerUserID,
		Provider:               normalized.Provider,
		Status:                 StatusPending,
		RequestedScopes:        normalized.RequestedScopes,
		StateHash:              normalized.StateHash,
		CodeVerifierCiphertext: normalized.CodeVerifierCiphertext,
		CreatedAt:              normalized.CreatedAt,
		UpdatedAt:              normalized.CreatedAt,
		ExpiresAt:              normalized.ExpiresAt,
	}, nil
}

// Complete marks a pending connect session completed at the given time.
func Complete(session Session, completedAt time.Time) (Session, error) {
	if session.ID == "" {
		return Session{}, ErrEmptyID
	}
	if session.OwnerUserID == "" {
		return Session{}, ErrEmptyOwnerUserID
	}
	if session.Status != StatusPending {
		return Session{}, errors.New("connect session is not pending")
	}
	completedAt = completedAt.UTC()
	session.Status = StatusCompleted
	session.CompletedAt = &completedAt
	session.UpdatedAt = completedAt
	return session, nil
}
