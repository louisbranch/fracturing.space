package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/public"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/publicprofile"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules/settings"
)

// DefaultPublicModules returns stable public web2 modules.
func DefaultPublicModules() []Module {
	return []Module{
		public.New(),
		discovery.New(),
		publicprofile.New(),
	}
}

// ExperimentalPublicModules returns opt-in public modules that are still scaffolded.
func ExperimentalPublicModules() []Module {
	return []Module{}
}

// DefaultProtectedModules returns stable authenticated web2 modules.
func DefaultProtectedModules(deps module.Dependencies) []Module {
	return []Module{
		settings.NewWithGateway(settings.NewGRPCGateway(deps)),
		campaigns.NewStableWithGateway(campaigns.NewGRPCGateway(deps)),
	}
}

// DefaultProtectedModulesWithExperimentalCampaignRoutes returns protected modules with experimental campaign route exposure.
func DefaultProtectedModulesWithExperimentalCampaignRoutes(deps module.Dependencies) []Module {
	return []Module{
		settings.NewWithGateway(settings.NewGRPCGateway(deps)),
		campaigns.NewWithGateway(campaigns.NewGRPCGateway(deps)),
	}
}

// ExperimentalProtectedModules returns opt-in authenticated modules that are still scaffolded.
func ExperimentalProtectedModules(deps module.Dependencies) []Module {
	return []Module{
		notifications.New(),
		profile.New(),
	}
}
