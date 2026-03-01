package app

import (
	"context"

	"golang.org/x/text/language"
)

const DegradedDependencySocialProfile = "social.profile"
const DegradedDependencyGameCampaigns = "game.campaigns"

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

// Gateway loads dashboard snapshot data for one user.
type Gateway interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error)
}

// Service exposes dashboard orchestration methods used by transport handlers.
type Service interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardView, error)
}
