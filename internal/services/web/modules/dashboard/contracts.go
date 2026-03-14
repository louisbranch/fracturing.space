package dashboard

import (
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
)

const degradedDependencySocialProfile = dashboardapp.DegradedDependencySocialProfile
const degradedDependencyGameCampaigns = dashboardapp.DegradedDependencyGameCampaigns
const degradedDependencyGameSessions = dashboardapp.DegradedDependencyGameSessions

// ServiceHealthEntry is the transport-facing alias for dashboard app health entries.
type ServiceHealthEntry = dashboardapp.ServiceHealthEntry

// DashboardView is the transport-facing alias for dashboard app view model.
type DashboardView = dashboardapp.DashboardView

// DashboardSnapshot is the transport-facing alias for dashboard app snapshot model.
type DashboardSnapshot = dashboardapp.DashboardSnapshot

// ActiveSessionItem is the transport-facing alias for dashboard active-session rows.
type ActiveSessionItem = dashboardapp.ActiveSessionItem

// DashboardGateway is the transport-facing alias for dashboard app gateway contract.
type DashboardGateway = dashboardapp.Gateway

// UserHubClient alias keeps root constructor/test seams stable.
type UserHubClient = dashboardgateway.UserHubClient
