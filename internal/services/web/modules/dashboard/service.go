package dashboard

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

const degradedDependencySocialProfile = "social.profile"
const degradedDependencyGameCampaigns = "game.campaigns"

// DashboardView is the web-dashboard view model derived from userhub state.
type DashboardView struct {
	ShowPendingProfileBlock bool
	ShowAdventureBlock      bool
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
	LoadDashboard(context.Context, string, commonv1.Locale) (DashboardSnapshot, error)
}

type service struct {
	readGateway DashboardGateway
}

type unavailableGateway struct{}

func (unavailableGateway) LoadDashboard(context.Context, string, commonv1.Locale) (DashboardSnapshot, error) {
	return DashboardSnapshot{}, nil
}

func newService(gateway DashboardGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{readGateway: gateway}
}

func (s service) loadDashboard(ctx context.Context, userID string, locale commonv1.Locale) (DashboardView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DashboardView{}, nil
	}
	snapshot, err := s.readGateway.LoadDashboard(ctx, userID, locale)
	if err != nil {
		return DashboardView{}, nil
	}
	if hasDegradedDependency(snapshot.DegradedDependencies, degradedDependencySocialProfile) {
		return DashboardView{}, nil
	}
	showAdventureBlock := false
	if !hasDegradedDependency(snapshot.DegradedDependencies, degradedDependencyGameCampaigns) {
		showAdventureBlock = !snapshot.HasDraftOrActiveCampaign && !snapshot.CampaignsHasMore
	}
	return DashboardView{
		ShowPendingProfileBlock: snapshot.NeedsProfileCompletion,
		ShowAdventureBlock:      showAdventureBlock,
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
