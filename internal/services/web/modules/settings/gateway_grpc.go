package settings

import (
	"context"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewGRPCGateway builds the production settings gateway from shared dependencies.
func NewGRPCGateway(deps module.Dependencies) SettingsGateway {
	if deps.SocialClient == nil && deps.AccountClient == nil && deps.CredentialClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{
		socialClient:     deps.SocialClient,
		accountClient:    deps.AccountClient,
		credentialClient: deps.CredentialClient,
	}
}

type grpcGateway struct {
	socialClient     module.SocialClient
	accountClient    module.AccountClient
	credentialClient module.CredentialClient
}

func (g grpcGateway) LoadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	if g.socialClient == nil {
		return SettingsProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return SettingsProfile{}, err
	}
	resp, err := g.socialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: resolvedUserID})
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
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
		Bio:           strings.TrimSpace(profile.GetBio()),
	}, nil
}

func (g grpcGateway) SaveProfile(ctx context.Context, userID string, profile SettingsProfile) error {
	if g.socialClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.social_service_is_not_configured", "social service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return err
	}
	_, err = g.socialClient.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{
		UserId:        resolvedUserID,
		Username:      profile.Username,
		Name:          profile.Name,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
		Bio:           profile.Bio,
	})
	return err
}

func (g grpcGateway) LoadLocale(ctx context.Context, userID string) (commonv1.Locale, error) {
	if g.accountClient == nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, err
	}
	resp, err := g.accountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: resolvedUserID})
	if err != nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, err
	}
	if resp == nil || resp.GetProfile() == nil {
		return commonv1.Locale_LOCALE_EN_US, nil
	}
	return resp.GetProfile().GetLocale(), nil
}

func (g grpcGateway) SaveLocale(ctx context.Context, userID string, locale commonv1.Locale) error {
	if g.accountClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.account_service_client_is_not_configured", "account service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return err
	}
	_, err = g.accountClient.UpdateProfile(ctx, &authv1.UpdateProfileRequest{UserId: resolvedUserID, Locale: locale})
	return err
}

func (g grpcGateway) ListAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error) {
	if g.credentialClient == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.credential_service_client_is_not_configured", "credential service client is not configured")
	}
	if _, err := requireUserID(userID); err != nil {
		return nil, err
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
	if _, err := requireUserID(userID); err != nil {
		return err
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
	if _, err := requireUserID(userID); err != nil {
		return err
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
