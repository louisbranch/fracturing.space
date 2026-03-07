package app

import (
	"context"
	"log"
	"strings"

	"golang.org/x/text/language"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	readGateway    Gateway
	logger         *log.Logger
	healthProvider HealthProvider
}

// NewService constructs a dashboard service with fail-closed gateway defaults.
func NewService(gateway Gateway, logger *log.Logger, health HealthProvider) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	if logger == nil {
		logger = log.Default()
	}
	return service{readGateway: gateway, logger: logger, healthProvider: health}
}

// LoadDashboard loads the package state needed for this request path.
func (s service) LoadDashboard(ctx context.Context, userID string, locale language.Tag) (DashboardView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DashboardView{}, nil
	}
	snapshot, err := s.readGateway.LoadDashboard(ctx, userID, locale)
	if err != nil {
		s.logger.Printf("dashboard: load failed for user %s: %v", userID, err)
		return DashboardView{}, nil
	}
	if HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencySocialProfile) {
		s.logger.Printf("dashboard: degraded dependency %s for user %s", DegradedDependencySocialProfile, userID)
		return DashboardView{}, nil
	}
	showAdventureBlock := false
	if !HasDegradedDependency(snapshot.DegradedDependencies, DegradedDependencyGameCampaigns) {
		showAdventureBlock = !snapshot.HasDraftOrActiveCampaign && !snapshot.CampaignsHasMore
	}
	var health []ServiceHealthEntry
	if s.healthProvider != nil {
		health = s.healthProvider(ctx)
	}
	return DashboardView{
		ShowPendingProfileBlock: snapshot.NeedsProfileCompletion,
		ShowAdventureBlock:      showAdventureBlock,
		ServiceHealth:           health,
	}, nil
}

// HasDegradedDependency reports whether a degraded dependency marker is present.
func HasDegradedDependency(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
