package dashboard

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

// NewGRPCGateway builds the production dashboard gateway from shared dependencies.
func NewGRPCGateway(deps module.Dependencies) DashboardGateway {
	if deps.UserHubClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: deps.UserHubClient}
}

type grpcGateway struct {
	client module.UserHubClient
}

func (g grpcGateway) LoadDashboard(ctx context.Context, userID string, locale commonv1.Locale) (DashboardSnapshot, error) {
	if g.client == nil {
		return DashboardSnapshot{}, nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DashboardSnapshot{}, nil
	}
	resp, err := g.client.GetDashboard(
		grpcauthctx.WithUserID(ctx, userID),
		&userhubv1.GetDashboardRequest{Locale: platformi18n.NormalizeLocale(locale)},
	)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	if resp == nil {
		return DashboardSnapshot{}, nil
	}
	return DashboardSnapshot{
		NeedsProfileCompletion: resp.GetUser().GetNeedsProfileCompletion(),
		DegradedDependencies:   normalizedDependencies(resp.GetMetadata().GetDegradedDependencies()),
	}, nil
}

func normalizedDependencies(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
