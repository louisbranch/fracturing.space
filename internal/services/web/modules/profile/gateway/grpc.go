package gateway

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// SocialClient exposes social profile lookup operations needed by profile pages.
type SocialClient interface {
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error)
}

// GRPCGateway maps gRPC social responses into app-layer profile contracts.
type GRPCGateway struct {
	Client SocialClient
}

// NewGRPCGateway builds a profile gateway backed by gRPC social client calls.
func NewGRPCGateway(client SocialClient) profileapp.Gateway {
	if client == nil {
		return profileapp.NewUnavailableGateway()
	}
	return GRPCGateway{Client: client}
}

// LookupUserProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) LookupUserProfile(ctx context.Context, req profileapp.LookupUserProfileRequest) (profileapp.LookupUserProfileResponse, error) {
	if g.Client == nil {
		return profileapp.LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
	}
	resp, err := g.Client.LookupUserProfile(ctx, &socialv1.LookupUserProfileRequest{Username: strings.TrimSpace(req.Username)})
	if err != nil {
		return profileapp.LookupUserProfileResponse{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackMessage: "social service is unavailable",
		})
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return profileapp.LookupUserProfileResponse{}, nil
	}
	profile := resp.GetUserProfile()
	return profileapp.LookupUserProfileResponse{
		UserID:        strings.TrimSpace(profile.GetUserId()),
		Username:      strings.TrimSpace(profile.GetUsername()),
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      pronouns.FromProto(profile.GetPronouns()),
		Bio:           strings.TrimSpace(profile.GetBio()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}, nil
}
