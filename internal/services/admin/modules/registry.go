package modules

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/scenarios"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/status"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/systems"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/users"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
)

// BuildInput carries dependencies required to build module sets.
type BuildInput struct {
	Base     modulehandler.Base
	GRPCAddr string

	// Individual gRPC clients used by modules.
	AuthClient               authv1.AuthServiceClient
	CampaignClient           statev1.CampaignServiceClient
	CharacterClient          statev1.CharacterServiceClient
	ParticipantClient        statev1.ParticipantServiceClient
	InviteClient             invitev1.InviteServiceClient
	SessionClient            statev1.SessionServiceClient
	EventClient              statev1.EventServiceClient
	StatisticsClient         statev1.StatisticsServiceClient
	SystemClient             statev1.SystemServiceClient
	DaggerheartContentClient daggerheartv1.DaggerheartContentServiceClient
	StatusClient             statusv1.StatusServiceClient
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
	input.ensureClients()
	return BuildOutput{Modules: []Module{
		dashboard.New(dashboard.NewHandlers(
			input.Base,
			input.StatisticsClient,
			input.SystemClient,
			input.AuthClient,
			input.CampaignClient,
			input.EventClient,
		)),
		campaigns.New(campaigns.NewHandlers(
			input.Base,
			input.CampaignClient,
			input.CharacterClient,
			input.ParticipantClient,
			input.InviteClient,
			input.SessionClient,
			input.EventClient,
			input.AuthClient,
		)),
		systems.New(systems.NewHandlers(input.Base, input.SystemClient)),
		catalog.New(catalog.NewHandlers(input.Base, input.DaggerheartContentClient)),
		icons.New(icons.NewHandlers(input.Base)),
		users.New(users.NewHandlers(input.Base, input.AuthClient, input.InviteClient)),
		scenarios.New(scenarios.NewHandlers(input.Base, input.GRPCAddr, input.EventClient, input.CampaignClient)),
		status.New(status.NewHandlers(input.Base, input.StatusClient)),
	}}
}
