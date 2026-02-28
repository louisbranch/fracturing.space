package settings

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SocialClient exposes profile lookup and mutation operations.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error)
	SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error)
}

// AccountClient exposes account profile read/update operations.
type AccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
	UpdateProfile(context.Context, *authv1.UpdateProfileRequest, ...grpc.CallOption) (*authv1.UpdateProfileResponse, error)
}

// CredentialClient exposes AI credential listing and mutation operations.
type CredentialClient interface {
	ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error)
	CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error)
	RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error)
}

// NewGRPCGateway builds the production settings gateway from the required clients.
// All three clients are required — a partial set would report healthy while
// individual settings pages 503.
func NewGRPCGateway(socialClient SocialClient, accountClient AccountClient, credentialClient CredentialClient) SettingsGateway {
	if socialClient == nil || accountClient == nil || credentialClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{
		socialClient:     socialClient,
		accountClient:    accountClient,
		credentialClient: credentialClient,
	}
}

type grpcGateway struct {
	socialClient     SocialClient
	accountClient    AccountClient
	credentialClient CredentialClient
}

func (g grpcGateway) LoadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	if g.socialClient == nil {
		return SettingsProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	resp, err := g.socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return SettingsProfile{}, nil
		}
		return SettingsProfile{}, err
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return SettingsProfile{}, nil
	}
	profile := resp.GetUserProfile()
	return SettingsProfile{
		Username:      strings.TrimSpace(profile.GetUsername()),
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      pronouns.FromProto(profile.GetPronouns()),
		Bio:           strings.TrimSpace(profile.GetBio()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}, nil
}

func (g grpcGateway) SaveProfile(ctx context.Context, userID string, profile SettingsProfile) error {
	if g.socialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	_, err := g.socialClient.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{
		UserId:        userID,
		Username:      profile.Username,
		Name:          profile.Name,
		Pronouns:      pronouns.ToProto(profile.Pronouns),
		Bio:           profile.Bio,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
	})
	return err
}

func (g grpcGateway) LoadLocale(ctx context.Context, userID string) (string, error) {
	if g.accountClient == nil {
		return "", apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	resp, err := g.accountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetProfile() == nil {
		return string(settingsLocaleEnUS), nil
	}
	return mapSettingsLocaleFromProto(resp.GetProfile().GetLocale()), nil
}

func (g grpcGateway) SaveLocale(ctx context.Context, userID string, locale string) error {
	if g.accountClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	_, err := g.accountClient.UpdateProfile(ctx, &authv1.UpdateProfileRequest{UserId: userID, Locale: mapSettingsLocaleToProto(locale)})
	return err
}

func (g grpcGateway) ListAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error) {
	if g.credentialClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	resp, err := g.credentialClient.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 50})
	if err != nil {
		return nil, err
	}
	keys := make([]SettingsAIKey, 0, len(resp.GetCredentials()))
	for _, credential := range resp.GetCredentials() {
		if credential == nil {
			continue
		}
		credentialID := strings.TrimSpace(credential.GetId())
		statusValue := credential.GetStatus()
		safeCredentialID := credentialID
		canRevoke := credentialID != "" && statusValue == aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
		if !isSafeCredentialPathID(credentialID) {
			safeCredentialID = ""
			canRevoke = false
		}
		keys = append(keys, SettingsAIKey{
			ID:        safeCredentialID,
			Label:     strings.TrimSpace(credential.GetLabel()),
			Provider:  providerDisplayLabel(credential.GetProvider()),
			Status:    credentialStatusDisplayLabel(statusValue),
			CreatedAt: formatProtoTimestamp(credential.GetCreatedAt()),
			RevokedAt: formatProtoTimestamp(credential.GetRevokedAt()),
			CanRevoke: canRevoke,
		})
	}
	return keys, nil
}

func (g grpcGateway) CreateAIKey(ctx context.Context, userID string, label string, secret string) error {
	if g.credentialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	_, err := g.credentialClient.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    label,
		Secret:   secret,
	})
	return err
}

func (g grpcGateway) RevokeAIKey(ctx context.Context, userID string, credentialID string) error {
	if g.credentialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	_, err := g.credentialClient.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: credentialID})
	return err
}

func providerDisplayLabel(provider aiv1.Provider) string {
	switch provider {
	case aiv1.Provider_PROVIDER_OPENAI:
		return "OpenAI"
	default:
		return "Unknown"
	}
}

func credentialStatusDisplayLabel(statusValue aiv1.CredentialStatus) string {
	switch statusValue {
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE:
		return "Active"
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED:
		return "Revoked"
	default:
		return "Unspecified"
	}
}

func formatProtoTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return "-"
	}
	if err := value.CheckValid(); err != nil {
		return "-"
	}
	return value.AsTime().UTC().Format("2006-01-02 15:04 UTC")
}

func mapSettingsLocaleToProto(locale string) commonv1.Locale {
	s := normalizeSettingsLocale(settingsLocale(locale))
	switch s {
	case settingsLocalePtBR:
		return commonv1.Locale_LOCALE_PT_BR
	case settingsLocaleEnUS:
		return commonv1.Locale_LOCALE_EN_US
	default:
		return commonv1.Locale_LOCALE_EN_US
	}
}

func mapSettingsLocaleFromProto(locale commonv1.Locale) string {
	switch locale {
	case commonv1.Locale_LOCALE_PT_BR:
		return string(settingsLocalePtBR)
	default:
		return string(settingsLocaleEnUS)
	}
}
