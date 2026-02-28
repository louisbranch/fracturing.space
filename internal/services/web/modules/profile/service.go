package profile

import (
	"context"
	"strings"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// Profile stores one public profile page payload.
type Profile struct {
	Username  string
	Name      string
	Pronouns  string
	Bio       string
	AvatarURL string
}

// LookupUserProfileRequest represents a domain request to load one user profile.
type LookupUserProfileRequest struct {
	Username string
}

// LookupUserProfileResponse stores the minimal profile fields the web module needs.
type LookupUserProfileResponse struct {
	Username      string
	UserID        string
	Name          string
	Pronouns      string
	Bio           string
	AvatarSetID   string
	AvatarAssetID string
}

// ProfileGateway abstracts profile lookup operations behind domain types.
type ProfileGateway interface {
	LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error)
}

type service struct {
	assetBaseURL string
	gateway      ProfileGateway
}

type unavailableGateway struct{}

const profileNotFoundMessage = "public profile not found"

func newService(gateway ProfileGateway, assetBaseURL string) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway, assetBaseURL: strings.TrimSpace(assetBaseURL)}
}

func (s service) loadProfile(ctx context.Context, username string) (Profile, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return Profile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
	}

	resp, err := s.gateway.LookupUserProfile(ctx, LookupUserProfileRequest{Username: username})
	if err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(resp.Username) == "" {
		return Profile{}, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage)
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

func (unavailableGateway) LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	return LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
}
