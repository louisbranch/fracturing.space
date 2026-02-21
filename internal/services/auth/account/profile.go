package account

import (
	"errors"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

var ErrEmptyUserID = errors.New("user id is required")

// Profile captures display/profile metadata for a user account.
type Profile struct {
	UserID    string
	Name      string
	Locale    commonv1.Locale
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileInput is the mutable payload used to create or update a profile.
type ProfileInput struct {
	UserID string
	Name   string
	Locale commonv1.Locale
}

// NormalizeProfileInput trims strings and normalizes locale.
func NormalizeProfileInput(input ProfileInput) ProfileInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Name = strings.TrimSpace(input.Name)
	input.Locale = i18n.NormalizeLocale(input.Locale)
	return input
}

// NewProfile validates and builds a full profile from input.
func NewProfile(input ProfileInput, now func() time.Time) (Profile, error) {
	normalized := NormalizeProfileInput(input)
	if normalized.UserID == "" {
		return Profile{}, ErrEmptyUserID
	}

	if now == nil {
		now = time.Now
	}
	nowUTC := func() time.Time { return now().UTC() }

	return Profile{
		UserID:    normalized.UserID,
		Name:      normalized.Name,
		Locale:    normalized.Locale,
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}, nil
}
