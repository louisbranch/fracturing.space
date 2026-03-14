// Package user provides auth user management.
package user

import (
	"fmt"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	authusername "github.com/louisbranch/fracturing.space/internal/services/auth/username"
)

var (
	// ErrEmptyUsername indicates a missing username.
	ErrEmptyUsername = apperrors.New(apperrors.CodeUserEmptyUsername, "Username is required.")
	// ErrInvalidUsername indicates a username that does not match auth policy.
	ErrInvalidUsername = apperrors.New(apperrors.CodeUserInvalidUsername, "Username must match the required format.")
)

// User represents an authenticated identity record.
type User struct {
	ID                        string
	Username                  string
	Locale                    commonv1.Locale
	RecoveryCodeHash          string
	RecoveryReservedSessionID string
	RecoveryReservedUntil     *time.Time
	RecoveryCodeUpdatedAt     time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// CreateUserInput describes the metadata needed to create a user.
type CreateUserInput struct {
	Username string
	Locale   commonv1.Locale
}

// ValidateUsername enforces canonical username constraints for auth identity.
func ValidateUsername(s string) error {
	if _, err := authusername.Canonicalize(s); err != nil {
		return ErrInvalidUsername
	}
	return nil
}

// CreateUser creates a durable user identity from validated input.
func CreateUser(input CreateUserInput, now func() time.Time, idGenerator func() (string, error)) (User, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateUserInput(input)
	if err != nil {
		return User{}, err
	}

	userID, err := idGenerator()
	if err != nil {
		return User{}, fmt.Errorf("Generate user ID: %w", err)
	}

	createdAt := now().UTC()
	return User{
		ID:                    userID,
		Username:              normalized.Username,
		Locale:                normalized.Locale,
		RecoveryCodeUpdatedAt: createdAt,
		CreatedAt:             createdAt,
		UpdatedAt:             createdAt,
	}, nil
}

// NormalizeCreateUserInput trims and normalizes input before validation.
func NormalizeCreateUserInput(input CreateUserInput) (CreateUserInput, error) {
	if input.Username == "" {
		return CreateUserInput{}, ErrEmptyUsername
	}
	canonicalUsername, err := authusername.Canonicalize(input.Username)
	if err != nil {
		return CreateUserInput{}, ErrInvalidUsername
	}
	input.Username = canonicalUsername
	input.Locale = platformi18n.NormalizeLocale(input.Locale)
	return input, nil
}
