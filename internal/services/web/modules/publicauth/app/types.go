package app

import (
	"context"
	"encoding/json"
)

// Gateway abstracts authentication operations behind domain types.
type Gateway interface {
	// BeginAccountRegistration starts account creation and first-passkey enrollment.
	BeginAccountRegistration(ctx context.Context, username string) (PasskeyChallenge, error)
	// FinishAccountRegistration completes account creation and returns the signed-in session.
	FinishAccountRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error)
	// BeginPasskeyLogin starts a username-scoped passkey login flow.
	BeginPasskeyLogin(ctx context.Context, username string) (PasskeyChallenge, error)
	// FinishPasskeyLogin completes login and returns the user ID.
	FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage) (string, error)
	// CreateWebSession creates a session for the given user, returning the session ID.
	CreateWebSession(ctx context.Context, userID string) (string, error)
	// HasValidWebSession checks whether a session exists and is valid.
	HasValidWebSession(ctx context.Context, sessionID string) bool
	// RevokeWebSession invalidates a web session.
	RevokeWebSession(ctx context.Context, sessionID string) error
}

// Service exposes publicauth orchestration methods used by transport handlers.
type Service interface {
	HealthBody() string
	PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error)
	PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error)
	PasskeyRegisterStart(ctx context.Context, username string) (PasskeyRegisterResult, error)
	PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error)
	HasValidWebSession(ctx context.Context, sessionID string) bool
	RevokeWebSession(ctx context.Context, sessionID string) error
}

// PasskeyChallenge holds the session and public key from a passkey begin operation.
type PasskeyChallenge struct {
	SessionID string
	PublicKey json.RawMessage
}

// PasskeyRegisterResult stores passkey registration start data.
type PasskeyRegisterResult struct {
	SessionID string
	PublicKey json.RawMessage
}

// PasskeyFinish stores passkey finish results.
type PasskeyFinish struct {
	SessionID    string
	UserID       string
	RecoveryCode string
}
