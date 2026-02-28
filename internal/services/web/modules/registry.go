package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/public"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
)

// DefaultPublicModules returns stable public web modules.
func DefaultPublicModules() []Module {
	return []Module{
		public.New(),
		discovery.New(),
		profile.New(),
	}
}

// ExperimentalPublicModules returns opt-in public modules that are still scaffolded.
func ExperimentalPublicModules() []Module {
	return []Module{}
}

// DefaultProtectedModules returns stable authenticated web modules.
func DefaultProtectedModules(deps module.Dependencies) []Module {
	return []Module{
		dashboard.NewWithGateway(dashboard.NewGRPCGateway(deps)),
		settings.NewWithGateway(settings.NewGRPCGateway(deps)),
		campaigns.NewStableWithGateway(campaigns.NewGRPCGateway(deps)),
		notifications.NewWithGateway(notifications.NewGRPCGateway(deps)),
	}
}

// DefaultProtectedModulesWithExperimentalCampaignRoutes returns protected modules with experimental campaign route exposure.
func DefaultProtectedModulesWithExperimentalCampaignRoutes(deps module.Dependencies) []Module {
	return []Module{
		dashboard.NewWithGateway(dashboard.NewGRPCGateway(deps)),
		settings.NewWithGateway(settings.NewGRPCGateway(deps)),
		campaigns.NewWithGateway(campaigns.NewGRPCGateway(deps)),
		notifications.NewWithGateway(notifications.NewGRPCGateway(deps)),
	}
}

// ExperimentalProtectedModules returns opt-in authenticated modules that are still scaffolded.
func ExperimentalProtectedModules(_ module.Dependencies) []Module {
	return []Module{}
}
