package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New(errors.CodeNotFound, "Record not found.")

// UserStore owns durable user identity, the core join point used by game and admin.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
	GetUserByUsername(ctx context.Context, username string) (user.User, error)
	ListUsers(ctx context.Context, pageSize int, pageToken string) (UserPage, error)
}

// UserPage is a cursor page of users for admin-facing browsing and audits.
type UserPage struct {
	Users         []user.User
	NextPageToken string
}

// RegistrationSession stores pending username signup state until WebAuthn completes.
type RegistrationSession struct {
	ID               string
	UserID           string
	Username         string
	Locale           string
	RecoveryCodeHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// RecoverySession stores a narrow recovery flow scoped to replacement passkey enrollment.
type RecoverySession struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// PasskeyCredential stores a WebAuthn credential record linked to a user identity.
type PasskeyCredential struct {
	CredentialID   string
	UserID         string
	CredentialJSON string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
}

// PasskeySession stores pending WebAuthn challenge state for registration or login.
type PasskeySession struct {
	ID          string
	Kind        string
	UserID      string
	SessionJSON string
	ExpiresAt   time.Time
}

// WebSession stores durable authenticated web session state.
type WebSession struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// PasskeyStore persists WebAuthn credential and challenge state.
type PasskeyStore interface {
	PutPasskeyCredential(ctx context.Context, credential PasskeyCredential) error
	GetPasskeyCredential(ctx context.Context, credentialID string) (PasskeyCredential, error)
	ListPasskeyCredentials(ctx context.Context, userID string) ([]PasskeyCredential, error)
	DeletePasskeyCredential(ctx context.Context, credentialID string) error
	DeletePasskeyCredentialsByUser(ctx context.Context, userID string) error
	DeletePasskeyCredentialsByUserExcept(ctx context.Context, userID string, credentialID string) error
	PutPasskeySession(ctx context.Context, session PasskeySession) error
	GetPasskeySession(ctx context.Context, id string) (PasskeySession, error)
	DeletePasskeySession(ctx context.Context, id string) error
	DeleteExpiredPasskeySessions(ctx context.Context, now time.Time) error
	PutRegistrationSession(ctx context.Context, session RegistrationSession) error
	GetRegistrationSession(ctx context.Context, id string) (RegistrationSession, error)
	DeleteRegistrationSession(ctx context.Context, id string) error
	DeleteExpiredRegistrationSessions(ctx context.Context, now time.Time) error
	PutRecoverySession(ctx context.Context, session RecoverySession) error
	GetRecoverySession(ctx context.Context, id string) (RecoverySession, error)
	DeleteRecoverySession(ctx context.Context, id string) error
	DeleteExpiredRecoverySessions(ctx context.Context, now time.Time) error
}

// WebSessionStore persists durable authenticated web sessions.
type WebSessionStore interface {
	PutWebSession(ctx context.Context, session WebSession) error
	GetWebSession(ctx context.Context, id string) (WebSession, error)
	RevokeWebSession(ctx context.Context, id string, revokedAt time.Time) error
	RevokeWebSessionsByUser(ctx context.Context, userID string, revokedAt time.Time) error
	DeleteExpiredWebSessions(ctx context.Context, now time.Time) error
}

// AuthStatistics contains aggregate counts across auth data.
type AuthStatistics struct {
	UserCount int64
}

// StatisticsStore provides aggregate auth statistics.
type StatisticsStore interface {
	// GetAuthStatistics returns aggregate counts.
	// When since is nil, counts are for all time.
	GetAuthStatistics(ctx context.Context, since *time.Time) (AuthStatistics, error)
}

const (
	// IntegrationOutboxStatusPending is ready for leasing and processing.
	IntegrationOutboxStatusPending = "pending"
	// IntegrationOutboxStatusLeased is currently leased by one worker.
	IntegrationOutboxStatusLeased = "leased"
	// IntegrationOutboxStatusSucceeded finished successfully.
	IntegrationOutboxStatusSucceeded = "succeeded"
	// IntegrationOutboxStatusDead exhausted retries and needs operator action.
	IntegrationOutboxStatusDead = "dead"
)

// IntegrationOutboxEvent is one durable integration work item.
//
// It is produced by auth-authoritative flows and consumed by integration workers.
type IntegrationOutboxEvent struct {
	ID             string
	EventType      string
	PayloadJSON    string
	DedupeKey      string
	Status         string
	AttemptCount   int
	NextAttemptAt  time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	LastError      string
	ProcessedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IntegrationOutboxStore persists durable integration work items for workers.
type IntegrationOutboxStore interface {
	EnqueueIntegrationOutboxEvent(ctx context.Context, event IntegrationOutboxEvent) error
	GetIntegrationOutboxEvent(ctx context.Context, id string) (IntegrationOutboxEvent, error)
	LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]IntegrationOutboxEvent, error)
	MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error
	MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error
	MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error
}

// UserOutboxTransactionalStore persists user + outbox event in one write unit.
//
// This protects signup flows from partial writes when outbox persistence fails.
type UserOutboxTransactionalStore interface {
	PutUserWithIntegrationOutboxEvent(ctx context.Context, u user.User, event IntegrationOutboxEvent) error
}

// UserSignupTransactionalStore persists first-account signup state atomically.
//
// This protects passkey signup flows from partial writes across identity,
// initial passkey enrollment, initial web session issuance, and downstream
// bootstrap integration events.
type UserSignupTransactionalStore interface {
	PutUserPasskeyWithIntegrationOutboxEvent(ctx context.Context, u user.User, credential PasskeyCredential, session WebSession, event IntegrationOutboxEvent) error
}
