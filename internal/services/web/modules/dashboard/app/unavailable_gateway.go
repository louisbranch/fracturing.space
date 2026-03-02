package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed dashboard gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// IsGatewayHealthy reports whether a dashboard gateway is configured.
func IsGatewayHealthy(gateway Gateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// LoadDashboard loads the package state needed for this request path.
func (unavailableGateway) LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error) {
	return DashboardSnapshot{}, apperrors.E(apperrors.KindUnavailable, "dashboard service is not configured")
}
