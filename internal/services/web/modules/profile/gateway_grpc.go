package profile

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// SocialClient exposes the social profile operations needed by the profile module.
type SocialClient interface {
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error)
}

// NewGRPCGateway builds a ProfileGateway backed by gRPC social client calls.
func NewGRPCGateway(client SocialClient) ProfileGateway {
	if client == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: client}
}

type grpcGateway struct {
	client SocialClient
}

func (g grpcGateway) LookupUserProfile(ctx context.Context, req LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	if g.client == nil {
		return LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
	}
	resp, err := g.client.LookupUserProfile(ctx, &socialv1.LookupUserProfileRequest{Username: strings.TrimSpace(req.Username)})
	if err != nil {
		return LookupUserProfileResponse{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackMessage: "social service is unavailable",
		})
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return LookupUserProfileResponse{}, nil
	}
	profile := resp.GetUserProfile()
	return LookupUserProfileResponse{
		UserID:        strings.TrimSpace(profile.GetUserId()),
		Username:      strings.TrimSpace(profile.GetUsername()),
		Name:          strings.TrimSpace(profile.GetName()),
		Pronouns:      strings.TrimSpace(profile.GetPronouns()),
		Bio:           strings.TrimSpace(profile.GetBio()),
		AvatarSetID:   strings.TrimSpace(profile.GetAvatarSetId()),
		AvatarAssetID: strings.TrimSpace(profile.GetAvatarAssetId()),
	}, nil
}
