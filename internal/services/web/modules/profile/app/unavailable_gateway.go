package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed profile gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// IsGatewayHealthy reports whether a profile gateway is configured and usable.
func IsGatewayHealthy(gateway Gateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

func (unavailableGateway) LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	return LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
}
