package settings

import (
	"context"
	"strings"
	"unicode/utf8"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const userProfileNameMaxLength = 64

// SettingsProfile stores editable user profile settings.
type SettingsProfile struct {
	Username      string
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Pronouns      string
	Bio           string
}

// SettingsAIKey stores a credential row displayed in the AI keys page.
type SettingsAIKey struct {
	ID        string
	Label     string
	Provider  string
	Status    string
	CreatedAt string
	RevokedAt string
	CanRevoke bool
}

// SettingsGateway loads and updates settings data for web handlers.
type SettingsGateway interface {
	LoadProfile(context.Context, string) (SettingsProfile, error)
	SaveProfile(context.Context, string, SettingsProfile) error
	LoadLocale(context.Context, string) (commonv1.Locale, error)
	SaveLocale(context.Context, string, commonv1.Locale) error
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, string, string) error
	RevokeAIKey(context.Context, string, string) error
}

type service struct {
	gateway SettingsGateway
}

type staticGateway struct{}

type unavailableGateway struct{}

func (staticGateway) LoadProfile(context.Context, string) (SettingsProfile, error) {
	return SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
}

func (staticGateway) SaveProfile(context.Context, string, SettingsProfile) error {
	return nil
}

func (staticGateway) LoadLocale(context.Context, string) (commonv1.Locale, error) {
	return commonv1.Locale_LOCALE_EN_US, nil
}

func (staticGateway) SaveLocale(context.Context, string, commonv1.Locale) error {
	return nil
}

func (staticGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return []SettingsAIKey{}, nil
}

func (staticGateway) CreateAIKey(context.Context, string, string, string) error {
	return nil
}

func (staticGateway) RevokeAIKey(context.Context, string, string) error {
	return nil
}

func (unavailableGateway) LoadProfile(context.Context, string) (SettingsProfile, error) {
	return SettingsProfile{}, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) SaveProfile(context.Context, string, SettingsProfile) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) LoadLocale(context.Context, string) (commonv1.Locale, error) {
	return commonv1.Locale_LOCALE_UNSPECIFIED, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) SaveLocale(context.Context, string, commonv1.Locale) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) CreateAIKey(context.Context, string, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) RevokeAIKey(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func newService(gateway SettingsGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

func (s service) loadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return SettingsProfile{}, err
	}
	profile, err := s.gateway.LoadProfile(ctx, resolvedUserID)
	if err != nil {
		return SettingsProfile{}, err
	}
	profile.Username = strings.TrimSpace(profile.Username)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.AvatarSetID = strings.TrimSpace(profile.AvatarSetID)
	profile.AvatarAssetID = strings.TrimSpace(profile.AvatarAssetID)
	profile.Bio = strings.TrimSpace(profile.Bio)
	profile.Pronouns = strings.TrimSpace(profile.Pronouns)
	return profile, nil
}

func (s service) saveProfile(ctx context.Context, userID string, profile SettingsProfile) error {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return err
	}
	profile.Username = strings.TrimSpace(profile.Username)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.AvatarSetID = strings.TrimSpace(profile.AvatarSetID)
	profile.AvatarAssetID = strings.TrimSpace(profile.AvatarAssetID)
	profile.Bio = strings.TrimSpace(profile.Bio)
	profile.Pronouns = strings.TrimSpace(profile.Pronouns)

	if utf8.RuneCountInString(profile.Name) > userProfileNameMaxLength {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.user_profile.error_name_too_long", "name is too long")
	}
	return s.gateway.SaveProfile(ctx, resolvedUserID, profile)
}

func (s service) loadLocale(ctx context.Context, userID string) (commonv1.Locale, error) {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, err
	}
	locale, err := s.gateway.LoadLocale(ctx, resolvedUserID)
	if err != nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, err
	}
	return platformi18n.NormalizeLocale(locale), nil
}

func (s service) saveLocale(ctx context.Context, userID string, value string) error {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return err
	}
	locale, ok := platformi18n.ParseLocale(value)
	if !ok {
		return apperrors.EK(apperrors.KindInvalidInput, "error.http.invalid_locale", "locale is invalid")
	}
	return s.gateway.SaveLocale(ctx, resolvedUserID, locale)
}

func (s service) listAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error) {
	resolvedUserID, err := requireUserID(userID)
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
		normalized = append(normalized, key)
	}

	return normalized, nil
}

func (s service) createAIKey(ctx context.Context, userID string, label string, secret string) error {
	resolvedUserID, err := requireUserID(userID)
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

func (s service) revokeAIKey(ctx context.Context, userID string, credentialID string) error {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(credentialID) == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.ai_key_id_is_required", "credential id is required")
	}
	return s.gateway.RevokeAIKey(ctx, resolvedUserID, strings.TrimSpace(credentialID))
}

func requireUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", apperrors.EK(apperrors.KindUnauthorized, "error.web.message.user_id_is_required", "user id is required")
	}
	return userID, nil
}

func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}
