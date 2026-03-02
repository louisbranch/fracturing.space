package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// unavailableGateway defines an internal contract used at this web package boundary.
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

// LookupUserProfile centralizes this web behavior in one helper seam.
func (unavailableGateway) LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error) {
	return LookupUserProfileResponse{}, apperrors.E(apperrors.KindUnavailable, "social service client is not configured")
}
