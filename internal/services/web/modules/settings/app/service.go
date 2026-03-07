package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	gateway Gateway
}

// NewService constructs a settings service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

// LoadProfile loads the package state needed for this request path.
func (s service) LoadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return SettingsProfile{}, err
	}
	profile, err := s.gateway.LoadProfile(ctx, resolvedUserID)
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
	return s.gateway.SaveProfile(ctx, resolvedUserID, profile)
}

// LoadLocale loads the package state needed for this request path.
func (s service) LoadLocale(ctx context.Context, userID string) (string, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return "", err
	}
	locale, err := s.gateway.LoadLocale(ctx, resolvedUserID)
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
	return s.gateway.SaveLocale(ctx, resolvedUserID, locale)
}

// ListAIKeys returns the package view collection for this workflow.
func (s service) ListAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	keys, err := s.gateway.ListAIKeys(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		return []SettingsAIKey{}, nil
	}

	normalized := make([]SettingsAIKey, 0, len(keys))
	for _, key := range keys {
		normalized = append(normalized, normalizeSettingsAIKey(key))
	}

	return normalized, nil
}

// CreateAIKey executes package-scoped creation behavior for this flow.
func (s service) CreateAIKey(ctx context.Context, userID string, label string, secret string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	label = strings.TrimSpace(label)
	secret = strings.TrimSpace(secret)
	if label == "" || secret == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_keys.error_required", "label and secret are required")
	}
	return s.gateway.CreateAIKey(ctx, resolvedUserID, label, secret)
}

// RevokeAIKey applies this package workflow transition.
func (s service) RevokeAIKey(ctx context.Context, userID string, credentialID string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	resolvedCredentialID := strings.TrimSpace(credentialID)
	if resolvedCredentialID == "" {
		return apperrors.EK(
			apperrors.KindInvalidInput,
			"error.web.message.ai_key_id_is_required",
			"credential id is required",
		)
	}
	return s.gateway.RevokeAIKey(ctx, resolvedUserID, resolvedCredentialID)
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

// normalizeSettingsAIKey normalizes one credential row for stable template rendering.
func normalizeSettingsAIKey(key SettingsAIKey) SettingsAIKey {
	key.ID = strings.TrimSpace(key.ID)
	key.Label = strings.TrimSpace(key.Label)
	key.Provider = strings.TrimSpace(key.Provider)
	key.Status = strings.TrimSpace(key.Status)
	key.CreatedAt = strings.TrimSpace(key.CreatedAt)
	key.RevokedAt = strings.TrimSpace(key.RevokedAt)

	if key.Provider == "" {
		key.Provider = "Unknown"
	}
	if key.Status == "" {
		key.Status = "Unspecified"
	}
	if key.CreatedAt == "" {
		key.CreatedAt = "-"
	}
	if key.RevokedAt == "" {
		key.RevokedAt = "-"
	}
	if !isSafeCredentialPathID(key.ID) {
		key.ID = ""
		key.CanRevoke = false
	}
	return key
}
