// Package user provides auth user management.
package user

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

var (
	// ErrEmptyPrimaryEmail indicates a missing primary email.
	ErrEmptyPrimaryEmail = apperrors.New(apperrors.CodeUserEmptyPrimaryEmail, "primary email is required")
	// ErrInvalidPrimaryEmail indicates a primary email that does not match the required format.
	ErrInvalidPrimaryEmail = apperrors.New(apperrors.CodeUserInvalidPrimaryEmail, "primary email must be a valid email address")
)

// User represents an authenticated identity record.
type User struct {
	ID           string
	PrimaryEmail string
	Locale       commonv1.Locale
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateUserInput describes the metadata needed to create a user.
type CreateUserInput struct {
	PrimaryEmail string
	Locale       commonv1.Locale
}

// ValidatePrimaryEmail enforces canonical primary-email constraints used by joins, invites,
// and chat display across services.
func ValidatePrimaryEmail(s string) error {
	parsed, err := mail.ParseAddress(s)
	if err != nil || parsed.Name != "" || !strings.EqualFold(strings.TrimSpace(parsed.Address), strings.TrimSpace(s)) {
		return ErrInvalidPrimaryEmail
	}
	return nil
}

// CreateUser creates a durable user identity from validated input.
//
// The service layer treats this as the canonical point where untrusted primary
// email becomes a stable identity used by auth, admin, and game paths.
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
		ID:           userID,
		PrimaryEmail: normalized.PrimaryEmail,
		Locale:       normalized.Locale,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}, nil
}

// NormalizeCreateUserInput trims and normalizes input before validation.
func NormalizeCreateUserInput(input CreateUserInput) (CreateUserInput, error) {
	input.PrimaryEmail = strings.ToLower(strings.TrimSpace(input.PrimaryEmail))
	if input.PrimaryEmail == "" {
		return CreateUserInput{}, ErrEmptyPrimaryEmail
	}
	if err := ValidatePrimaryEmail(input.PrimaryEmail); err != nil {
		return CreateUserInput{}, err
	}
	input.Locale = platformi18n.NormalizeLocale(input.Locale)
	return input, nil
}
