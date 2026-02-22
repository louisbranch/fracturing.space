package storage

import (
	"context"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New(errors.CodeNotFound, "record not found")

// UserStore owns durable user identity, the core join point used by game and admin.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
	ListUsers(ctx context.Context, pageSize int, pageToken string) (UserPage, error)
}

// AccountProfile represents one row of user profile metadata.
type AccountProfile struct {
	UserID        string
	Name          string
	Locale        commonv1.Locale
	AvatarSetID   string
	AvatarAssetID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AccountProfileStore persists per-user profile metadata.
type AccountProfileStore interface {
	PutAccountProfile(ctx context.Context, profile AccountProfile) error
	GetAccountProfile(ctx context.Context, userID string) (AccountProfile, error)
}

// UserPage is a cursor page of users for admin-facing browsing and audits.
type UserPage struct {
	Users         []user.User
	NextPageToken string
}

// Contact represents one owner-scoped quick-lookup relationship.
type Contact struct {
	OwnerUserID   string
	ContactUserID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContactPage is a cursor page of contacts.
type ContactPage struct {
	Contacts      []Contact
	NextPageToken string
}

// ContactStore persists owner-scoped user contacts.
type ContactStore interface {
	PutContact(ctx context.Context, contact Contact) error
	GetContact(ctx context.Context, ownerUserID string, contactUserID string) (Contact, error)
	DeleteContact(ctx context.Context, ownerUserID string, contactUserID string) error
	ListContacts(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (ContactPage, error)
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

// UserEmail stores a verified contact address and verification lifecycle metadata.
type UserEmail struct {
	ID         string
	UserID     string
	Email      string
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MagicLink represents a single-use bootstrap token for one-time authentication.
type MagicLink struct {
	Token     string
	UserID    string
	Email     string
	PendingID string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// PasskeyStore persists WebAuthn credential and challenge state.
type PasskeyStore interface {
	PutPasskeyCredential(ctx context.Context, credential PasskeyCredential) error
	GetPasskeyCredential(ctx context.Context, credentialID string) (PasskeyCredential, error)
	ListPasskeyCredentials(ctx context.Context, userID string) ([]PasskeyCredential, error)
	DeletePasskeyCredential(ctx context.Context, credentialID string) error
	PutPasskeySession(ctx context.Context, session PasskeySession) error
	GetPasskeySession(ctx context.Context, id string) (PasskeySession, error)
	DeletePasskeySession(ctx context.Context, id string) error
	DeleteExpiredPasskeySessions(ctx context.Context, now time.Time) error
}

// EmailStore persists user contacts used for identity recovery and validation.
type EmailStore interface {
	PutUserEmail(ctx context.Context, email UserEmail) error
	GetUserEmailByEmail(ctx context.Context, email string) (UserEmail, error)
	ListUserEmailsByUser(ctx context.Context, userID string) ([]UserEmail, error)
	VerifyUserEmail(ctx context.Context, userID string, email string, verifiedAt time.Time) error
}

// MagicLinkStore persists one-time magic-link token state.
type MagicLinkStore interface {
	PutMagicLink(ctx context.Context, link MagicLink) error
	GetMagicLink(ctx context.Context, token string) (MagicLink, error)
	MarkMagicLinkUsed(ctx context.Context, token string, usedAt time.Time) error
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
