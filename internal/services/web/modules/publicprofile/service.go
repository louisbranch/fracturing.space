package publicprofile

import (
	"context"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicProfile stores one public profile page payload.
type PublicProfile struct {
	Username  string
	Name      string
	Bio       string
	AvatarURL string
}

type socialGateway interface {
	LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest) (*socialv1.LookupUserProfileResponse, error)
}

type service struct {
	assetBaseURL string
	gateway      socialGateway
}

type grpcGateway struct {
	client module.SocialClient
}

type unavailableGateway struct{}

const profileNotFoundMessage = "public profile not found"

func newService(gateway socialGateway, assetBaseURL string) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway, assetBaseURL: strings.TrimSpace(assetBaseURL)}
}

func newGRPCGateway(deps module.Dependencies) socialGateway {
	if deps.SocialClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: deps.SocialClient}
}

func (s service) loadProfile(ctx context.Context, username string) (PublicProfile, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return PublicProfile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
	}

	resp, err := s.gateway.LookupUserProfile(ctx, &socialv1.LookupUserProfileRequest{Username: username})
	if err != nil {
		if status.Code(err) == codes.InvalidArgument {
			return PublicProfile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
		}
		return PublicProfile{}, err
	}
	if resp == nil || resp.GetUserProfile() == nil {
		return PublicProfile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
	}

	profile := resp.GetUserProfile()
	resolvedUsername := strings.TrimSpace(profile.GetUsername())
	if resolvedUsername == "" {
		return PublicProfile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
	}
	entityID := strings.TrimSpace(profile.GetUserId())
	if entityID == "" {
		entityID = resolvedUsername
	}

	return PublicProfile{
		Username: resolvedUsername,
		Name:     strings.TrimSpace(profile.GetName()),
		Bio:      strings.TrimSpace(profile.GetBio()),
		AvatarURL: websupport.AvatarImageURL(
			s.assetBaseURL,
			"user",
			entityID,
			strings.TrimSpace(profile.GetAvatarSetId()),
			strings.TrimSpace(profile.GetAvatarAssetId()),
		),
	}, nil
}

func (g grpcGateway) LookupUserProfile(ctx context.Context, req *socialv1.LookupUserProfileRequest) (*socialv1.LookupUserProfileResponse, error) {
	if g.client == nil {
		return nil, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
	}
	resp, err := g.client.LookupUserProfile(ctx, req)
	if err == nil {
		return resp, nil
	}
	switch status.Code(err) {
	case codes.NotFound:
		return nil, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
	case codes.Unavailable:
		return nil, apperrors.E(apperrors.KindUnavailable, "social service is unavailable")
	default:
		return nil, err
	}
}

func (unavailableGateway) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest) (*socialv1.LookupUserProfileResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
}
