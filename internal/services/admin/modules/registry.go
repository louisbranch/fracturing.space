package modules

import (
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/scenarios"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/systems"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/users"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
)

// BuildInput carries dependencies required to build module sets.
type BuildInput struct {
	Base     modulehandler.Base
	GRPCAddr string
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
	return BuildOutput{Modules: []Module{
		dashboard.New(dashboard.NewService(input.Base)),
		campaigns.New(campaigns.NewService(input.Base)),
		systems.New(systems.NewService(input.Base)),
		catalog.New(catalog.NewService(input.Base)),
		icons.New(icons.NewService(input.Base)),
		users.New(users.NewService(input.Base)),
		scenarios.New(scenarios.NewService(input.Base, input.GRPCAddr)),
	}}
}
