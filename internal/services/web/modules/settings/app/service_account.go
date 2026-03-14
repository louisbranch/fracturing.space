package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// LoadProfile loads the package state needed for this request path.
func (s service) LoadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return SettingsProfile{}, err
	}
	profile, err := s.profileGateway.LoadProfile(ctx, resolvedUserID)
	if err != nil {
		return SettingsProfile{}, err
	}
	return normalizeSettingsProfile(profile), nil
}

// SaveProfile centralizes this web behavior in one helper seam.
func (s service) SaveProfile(ctx context.Context, userID string, profile SettingsProfile) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	profile = normalizeSettingsProfile(profile)
	if err := validateNameLength(profile.Name); err != nil {
		return err
	}
	return s.profileGateway.SaveProfile(ctx, resolvedUserID, profile)
}

// LoadLocale loads the package state needed for this request path.
func (s service) LoadLocale(ctx context.Context, userID string) (string, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return "", err
	}
	locale, err := s.localeGateway.LoadLocale(ctx, resolvedUserID)
	if err != nil {
		return "", err
	}
	return NormalizeLocale(locale), nil
}

// SaveLocale centralizes this web behavior in one helper seam.
func (s service) SaveLocale(ctx context.Context, userID string, value string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	locale, ok := ParseLocale(value)
	if !ok {
		return apperrors.EK(apperrors.KindInvalidInput, "error.http.invalid_locale", "locale is invalid")
	}
	return s.localeGateway.SaveLocale(ctx, resolvedUserID, locale)
}

// normalizeSettingsProfile centralizes profile field normalization before service flows.
func normalizeSettingsProfile(profile SettingsProfile) SettingsProfile {
	profile.Username = strings.TrimSpace(profile.Username)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.AvatarSetID = strings.TrimSpace(profile.AvatarSetID)
	profile.AvatarAssetID = strings.TrimSpace(profile.AvatarAssetID)
	profile.Bio = strings.TrimSpace(profile.Bio)
	profile.Pronouns = strings.TrimSpace(profile.Pronouns)
	return profile
}
