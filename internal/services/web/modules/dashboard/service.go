package dashboard

import (
	"context"
	"log"
	"strings"

	"golang.org/x/text/language"
)

const degradedDependencySocialProfile = "social.profile"
const degradedDependencyGameCampaigns = "game.campaigns"

// ServiceHealthEntry represents the availability status of a backend service group.
type ServiceHealthEntry struct {
	Label     string
	Available bool
}

// DashboardView is the web-dashboard view model derived from userhub state.
type DashboardView struct {
	ShowPendingProfileBlock bool
	ShowAdventureBlock      bool
	ServiceHealth           []ServiceHealthEntry
}

// DashboardSnapshot contains userhub dashboard fields used by web rendering logic.
type DashboardSnapshot struct {
	NeedsProfileCompletion   bool
	HasDraftOrActiveCampaign bool
	CampaignsHasMore         bool
	DegradedDependencies     []string
}

// DashboardGateway loads dashboard snapshot data for one user.
type DashboardGateway interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error)
}

type service struct {
	readGateway   DashboardGateway
	logger        *log.Logger
	serviceHealth []ServiceHealthEntry
}

func newService(gateway DashboardGateway, logger *log.Logger, health []ServiceHealthEntry) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	if logger == nil {
		logger = log.Default()
	}
	return service{readGateway: gateway, logger: logger, serviceHealth: health}
}

func (s service) loadDashboard(ctx context.Context, userID string, locale language.Tag) (DashboardView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DashboardView{}, nil
	}
	snapshot, err := s.readGateway.LoadDashboard(ctx, userID, locale)
	if err != nil {
		s.logger.Printf("dashboard: load failed for user %s: %v", userID, err)
		return DashboardView{}, nil
	}
	if hasDegradedDependency(snapshot.DegradedDependencies, degradedDependencySocialProfile) {
		s.logger.Printf("dashboard: degraded dependency %s for user %s", degradedDependencySocialProfile, userID)
		return DashboardView{}, nil
	}
	showAdventureBlock := false
	if !hasDegradedDependency(snapshot.DegradedDependencies, degradedDependencyGameCampaigns) {
		showAdventureBlock = !snapshot.HasDraftOrActiveCampaign && !snapshot.CampaignsHasMore
	}
	return DashboardView{
		ShowPendingProfileBlock: snapshot.NeedsProfileCompletion,
		ShowAdventureBlock:      showAdventureBlock,
		ServiceHealth:           s.serviceHealth,
	}, nil
}

func hasDegradedDependency(values []string, want string) bool {
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
