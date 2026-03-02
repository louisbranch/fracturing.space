package modules

import (
	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/campaigns"
	catalogmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/catalog"
	dashboardmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/dashboard"
	iconsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/icons"
	scenariosmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/scenarios"
	systemsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/systems"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/scenarios"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/systems"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/users"
)

// Service exposes admin behavior behind stable module contracts.
type Service interface {
	dashboardmodule.Service
	campaignsmodule.Service
	systemsmodule.Service
	catalogmodule.Service
	iconsmodule.Service
	scenariosmodule.Service
	users.Service
}

// BuildInput carries dependencies required to build module sets.
type BuildInput struct {
	Service Service
}

// BuildOutput contains composed module sets.
type BuildOutput struct {
	Modules []Module
}

// Registry builds the default admin module set.
type Registry struct{}

// NewRegistry returns the default admin module registry.
func NewRegistry() Registry { return Registry{} }

// Build composes module sets for admin.
func (Registry) Build(input BuildInput) BuildOutput {
	svc := input.Service
	return BuildOutput{Modules: []Module{
		dashboard.New(svc),
		campaigns.New(svc),
		systems.New(svc),
		catalog.New(svc),
		icons.New(svc),
		users.New(svc),
		scenarios.New(svc),
	}}
}
