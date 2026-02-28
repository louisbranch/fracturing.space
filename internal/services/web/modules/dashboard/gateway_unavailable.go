package dashboard

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

type unavailableGateway struct{}

func (unavailableGateway) LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error) {
	return DashboardSnapshot{}, apperrors.E(apperrors.KindUnavailable, "dashboard service is not configured")
}
