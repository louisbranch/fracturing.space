package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New(errors.CodeNotFound, "record not found")

// UserStore persists auth user records.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
	ListUsers(ctx context.Context, pageSize int, pageToken string) (UserPage, error)
}

// UserPage describes a page of user records.
type UserPage struct {
	Users         []user.User
	NextPageToken string
}

// PasskeyCredential stores a WebAuthn credential for a user.
type PasskeyCredential struct {
	CredentialID   string
	UserID         string
	CredentialJSON string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
}

// PasskeySession stores a WebAuthn registration or login session.
type PasskeySession struct {
	ID          string
	Kind        string
	UserID      string
	SessionJSON string
	ExpiresAt   time.Time
}

// UserEmail stores an email address tied to a user.
type UserEmail struct {
	ID         string
	UserID     string
	Email      string
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MagicLink represents a single-use magic link token.
type MagicLink struct {
	Token     string
	UserID    string
	Email     string
	PendingID string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// PasskeyStore persists WebAuthn credential and session data.
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

// EmailStore persists user email data.
type EmailStore interface {
	PutUserEmail(ctx context.Context, email UserEmail) error
	GetUserEmailByEmail(ctx context.Context, email string) (UserEmail, error)
	ListUserEmailsByUser(ctx context.Context, userID string) ([]UserEmail, error)
	VerifyUserEmail(ctx context.Context, userID string, email string, verifiedAt time.Time) error
}

// MagicLinkStore persists magic link tokens.
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
