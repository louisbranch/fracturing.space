package app

import (
	"context"
	"encoding/json"
)

// Gateway abstracts authentication operations behind domain types.
type Gateway interface {
	// BeginAccountRegistration starts account creation and first-passkey enrollment.
	BeginAccountRegistration(ctx context.Context, username string) (PasskeyChallenge, error)
	// CheckUsernameAvailability validates and checks username availability.
	CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error)
	// FinishAccountRegistration stages signup completion and returns the recovery code reveal.
	FinishAccountRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyRegistrationReveal, error)
	// AcknowledgeAccountRegistration activates a staged signup and returns the signed-in session.
	AcknowledgeAccountRegistration(ctx context.Context, sessionID string, pendingID string) (PasskeyFinish, error)
	// BeginPasskeyLogin starts a username-scoped passkey login flow.
	BeginPasskeyLogin(ctx context.Context, username string) (PasskeyChallenge, error)
	// FinishPasskeyLogin completes login and returns the user ID.
	FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (string, error)
	// BeginAccountRecovery verifies the recovery code and returns a recovery session.
	BeginAccountRecovery(ctx context.Context, username string, recoveryCode string) (string, error)
	// BeginRecoveryPasskeyRegistration starts replacement passkey enrollment for recovery.
	BeginRecoveryPasskeyRegistration(ctx context.Context, recoverySessionID string) (PasskeyChallenge, error)
	// FinishRecoveryPasskeyRegistration completes recovery and returns the signed-in session.
	FinishRecoveryPasskeyRegistration(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error)
	// CreateWebSession creates a session for the given user, returning the session ID.
	CreateWebSession(ctx context.Context, userID string) (string, error)
	// RevokeWebSession invalidates a web session.
	RevokeWebSession(ctx context.Context, sessionID string) error
}

// PageService exposes shell and redirect behavior used by page handlers.
type PageService interface {
	HealthBody() string
	ResolvePostAuthRedirect(pendingID string, nextPath string) string
}

// SessionService exposes session-backed redirect and logout behavior.
type SessionService interface {
	ResolvePostAuthRedirect(pendingID string, nextPath string) string
	RevokeWebSession(ctx context.Context, sessionID string) error
}

// PasskeyService exposes username, login, and signup ceremonies.
type PasskeyService interface {
	CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error)
	PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error)
	PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error)
	PasskeyRegisterStart(ctx context.Context, username string) (PasskeyRegisterResult, error)
	PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyRegistrationReveal, error)
	PasskeyRegisterAcknowledge(ctx context.Context, sessionID string, pendingID string) (PasskeyFinish, error)
}

// RecoveryService exposes account recovery ceremonies.
type RecoveryService interface {
	RecoveryStart(ctx context.Context, username string, recoveryCode string) (RecoveryChallenge, error)
	RecoveryFinish(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error)
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

// PasskeyRegistrationReveal stores the one-time recovery code shown before activation.
type PasskeyRegistrationReveal struct {
	RecoveryCode string
}

// UsernameAvailabilityState identifies the live validation outcome for signup input.
type UsernameAvailabilityState string

const (
	UsernameAvailabilityStateInvalid     UsernameAvailabilityState = "invalid"
	UsernameAvailabilityStateUnavailable UsernameAvailabilityState = "unavailable"
	UsernameAvailabilityStateAvailable   UsernameAvailabilityState = "available"
)

// UsernameAvailability stores live username validation state for signup UX.
type UsernameAvailability struct {
	CanonicalUsername string
	State             UsernameAvailabilityState
}

// RecoveryChallenge stores the combined recovery session and replacement-passkey begin state.
type RecoveryChallenge struct {
	RecoverySessionID string
	SessionID         string
	PublicKey         json.RawMessage
}

// PasskeyFinish stores passkey finish results.
type PasskeyFinish struct {
	SessionID    string
	UserID       string
	RecoveryCode string
}
