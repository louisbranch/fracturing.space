// Package user provides auth user management.
package user

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

var (
	// ErrEmptyUsername indicates a missing username.
	ErrEmptyUsername = apperrors.New(apperrors.CodeUserEmptyUsername, "username is required")
	// ErrInvalidUsername indicates a username that does not match the required format.
	ErrInvalidUsername = apperrors.New(apperrors.CodeUserInvalidUsername, "username must be 3-32 lowercase alphanumeric, dot, dash, or underscore characters")

	usernamePattern = regexp.MustCompile(`^[a-z0-9_.\-]{3,32}$`)
)

// User represents an authenticated identity record.
type User struct {
	ID        string
	Username  string
	Locale    commonv1.Locale
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateUserInput describes the metadata needed to create a user.
type CreateUserInput struct {
	Username string
	Locale   commonv1.Locale
}

// ValidateUsername checks that a username matches ^[a-z0-9_.-]{3,32}$.
func ValidateUsername(s string) error {
	if !usernamePattern.MatchString(s) {
		return ErrInvalidUsername
	}
	return nil
}

// CreateUser creates a new user with a generated ID and timestamps.
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
		return User{}, fmt.Errorf("generate user id: %w", err)
	}

	createdAt := now().UTC()
	return User{
		ID:        userID,
		Username:  normalized.Username,
		Locale:    normalized.Locale,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

// NormalizeCreateUserInput trims, lowercases, and validates user input metadata.
func NormalizeCreateUserInput(input CreateUserInput) (CreateUserInput, error) {
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	if input.Username == "" {
		return CreateUserInput{}, ErrEmptyUsername
	}
	if err := ValidateUsername(input.Username); err != nil {
		return CreateUserInput{}, err
	}
	input.Locale = platformi18n.NormalizeLocale(input.Locale)
	return input, nil
}
