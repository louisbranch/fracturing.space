package profile

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
)

// ProfileSummary is a transport-safe representation of the current profile.
type ProfileSummary struct {
	DisplayName string `json:"displayName"`
	Username    string `json:"username"`
}

// ProfileGateway loads profile data for web handlers.
type ProfileGateway interface {
	LoadProfile(context.Context) (ProfileSummary, error)
}

type service struct {
	gateway ProfileGateway
}

type staticGateway struct{}

type unavailableGateway struct{}

func (staticGateway) LoadProfile(context.Context) (ProfileSummary, error) {
	return ProfileSummary{DisplayName: "Adventurer", Username: "adventurer"}, nil
}

func (unavailableGateway) LoadProfile(context.Context) (ProfileSummary, error) {
	return ProfileSummary{}, apperrors.E(apperrors.KindUnavailable, "profile service is not configured")
}

func newService(gateway ProfileGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

func (s service) loadProfile(ctx context.Context) (ProfileSummary, error) {
	summary, err := s.gateway.LoadProfile(ctx)
	if err != nil {
		return ProfileSummary{}, err
	}
	if summary.DisplayName == "" {
		return ProfileSummary{}, apperrors.E(apperrors.KindNotFound, "profile not found")
	}
	return summary, nil
}
