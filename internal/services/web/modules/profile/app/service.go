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

// NewService constructs a profile service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

// LoadProfile loads the package state needed for this request path.
func (s service) LoadProfile(ctx context.Context, username string) (Profile, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return Profile{}, apperrors.E(apperrors.KindNotFound, ProfileNotFoundMessage)
	}

	resp, err := s.gateway.LookupUserProfile(ctx, LookupUserProfileRequest{Username: username})
	if err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(resp.Username) == "" {
		return Profile{}, apperrors.E(apperrors.KindNotFound, ProfileNotFoundMessage)
	}

	return Profile{
		Username:            strings.TrimSpace(resp.Username),
		UserID:              strings.TrimSpace(resp.UserID),
		Name:                strings.TrimSpace(resp.Name),
		Pronouns:            strings.TrimSpace(resp.Pronouns),
		Bio:                 strings.TrimSpace(resp.Bio),
		AvatarSetID:         strings.TrimSpace(resp.AvatarSetID),
		AvatarAssetID:       strings.TrimSpace(resp.AvatarAssetID),
		SocialProfileStatus: resp.SocialProfileStatus,
	}, nil
}
