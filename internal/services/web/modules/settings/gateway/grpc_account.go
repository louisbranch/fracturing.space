package gateway

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoadProfile loads the package state needed for this request path.
func (g GRPCGateway) LoadProfile(ctx context.Context, userID string) (settingsapp.SettingsProfile, error) {
	if g.SocialClient == nil {
		return settingsapp.SettingsProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	resp, err := g.SocialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp = nil
		} else {
			return settingsapp.SettingsProfile{}, err
		}
	}
	result := settingsapp.SettingsProfile{}
	if g.AccountClient != nil {
		accountResp, err := g.AccountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
		if err != nil {
			return settingsapp.SettingsProfile{}, err
		}
		if accountResp != nil && accountResp.GetProfile() != nil {
			result.Username = strings.TrimSpace(accountResp.GetProfile().GetUsername())
		}
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return result, nil
	}
	profile := resp.GetUserProfile()
	result.Name = strings.TrimSpace(profile.GetName())
	result.Pronouns = pronouns.FromProto(profile.GetPronouns())
	result.Bio = strings.TrimSpace(profile.GetBio())
	result.AvatarSetID = strings.TrimSpace(profile.GetAvatarSetId())
	result.AvatarAssetID = strings.TrimSpace(profile.GetAvatarAssetId())
	return result, nil
}

// SaveProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) SaveProfile(ctx context.Context, userID string, profile settingsapp.SettingsProfile) error {
	if g.SocialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	_, err := g.SocialClient.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{
		UserId:        userID,
		Name:          profile.Name,
		Pronouns:      pronouns.ToProto(profile.Pronouns),
		Bio:           profile.Bio,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
	})
	return err
}

// LoadLocale loads the package state needed for this request path.
func (g GRPCGateway) LoadLocale(ctx context.Context, userID string) (string, error) {
	if g.AccountClient == nil {
		return "", apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	resp, err := g.AccountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetProfile() == nil {
		return settingsapp.NormalizeLocale(""), nil
	}
	return mapSettingsLocaleFromProto(resp.GetProfile().GetLocale()), nil
}

// SaveLocale centralizes this web behavior in one helper seam.
func (g GRPCGateway) SaveLocale(ctx context.Context, userID string, locale string) error {
	if g.AccountClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	_, err := g.AccountClient.UpdateProfile(ctx, &authv1.UpdateProfileRequest{UserId: userID, Locale: mapSettingsLocaleToProto(locale)})
	return err
}

// mapSettingsLocaleToProto maps values across transport and domain boundaries.
func mapSettingsLocaleToProto(locale string) commonv1.Locale {
	s := settingsapp.NormalizeLocale(locale)
	switch s {
	case "pt-BR":
		return commonv1.Locale_LOCALE_PT_BR
	default:
		return commonv1.Locale_LOCALE_EN_US
	}
}

// mapSettingsLocaleFromProto maps values across transport and domain boundaries.
func mapSettingsLocaleFromProto(locale commonv1.Locale) string {
	switch locale {
	case commonv1.Locale_LOCALE_PT_BR:
		return "pt-BR"
	default:
		return "en-US"
	}
}
