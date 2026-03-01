package app

import (
	"context"
	"strings"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type service struct {
	assetBaseURL string
	gateway      Gateway
}

// NewService constructs a profile service with fail-closed gateway defaults.
func NewService(gateway Gateway, assetBaseURL string) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway, assetBaseURL: strings.TrimSpace(assetBaseURL)}
}

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

	entityID := strings.TrimSpace(resp.UserID)
	if entityID == "" {
		entityID = strings.TrimSpace(resp.Username)
	}

	return Profile{
		Username: strings.TrimSpace(resp.Username),
		Name:     strings.TrimSpace(resp.Name),
		Pronouns: strings.TrimSpace(resp.Pronouns),
		Bio:      strings.TrimSpace(resp.Bio),
		AvatarURL: websupport.AvatarImageURL(
			s.assetBaseURL,
			"user",
			entityID,
			strings.TrimSpace(resp.AvatarSetID),
			strings.TrimSpace(resp.AvatarAssetID),
		),
	}, nil
}
