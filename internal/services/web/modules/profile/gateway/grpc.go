package gateway

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// AuthClient resolves usernames to auth-owned account records.
type AuthClient interface {
	LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error)
}

// SocialClient exposes social profile lookup operations needed by profile pages.
type SocialClient interface {
	GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error)
}

// GRPCGateway maps gRPC auth/social responses into app-layer profile contracts.
type GRPCGateway struct {
	AuthClient   AuthClient
	SocialClient SocialClient
}

// NewGRPCGateway builds a profile gateway backed by gRPC auth and social client calls.
func NewGRPCGateway(authClient AuthClient, socialClient SocialClient) profileapp.Gateway {
	if authClient == nil {
		return profileapp.NewUnavailableGateway()
	}
	return GRPCGateway{AuthClient: authClient, SocialClient: socialClient}
}

// LookupUserProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) LookupUserProfile(ctx context.Context, req profileapp.LookupUserProfileRequest) (profileapp.LookupUserProfileResponse, error) {
	if g.AuthClient == nil {
		return profileapp.LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "auth service client is not configured")
	}

	authResp, err := g.AuthClient.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: strings.TrimSpace(req.Username)})
	if err != nil {
		return profileapp.LookupUserProfileResponse{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackMessage: "auth service is unavailable",
		})
	}
	if authResp == nil || authResp.GetUser() == nil {
		return profileapp.LookupUserProfileResponse{}, nil
	}

	account := authResp.GetUser()
	response := profileapp.LookupUserProfileResponse{
		UserID:   strings.TrimSpace(account.GetId()),
		Username: strings.TrimSpace(account.GetUsername()),
	}

	if g.SocialClient == nil || response.UserID == "" {
		return response, nil
	}
	socialResp, err := g.SocialClient.GetUserProfile(ctx, &socialv1.GetUserProfileRequest{UserId: response.UserID})
	if err != nil || socialResp == nil || socialResp.GetUserProfile() == nil {
		return response, nil
	}

	profile := socialResp.GetUserProfile()
	response.Name = strings.TrimSpace(profile.GetName())
	response.Pronouns = pronouns.FromProto(profile.GetPronouns())
	response.Bio = strings.TrimSpace(profile.GetBio())
	response.AvatarSetID = strings.TrimSpace(profile.GetAvatarSetId())
	response.AvatarAssetID = strings.TrimSpace(profile.GetAvatarAssetId())
	return response, nil
}
