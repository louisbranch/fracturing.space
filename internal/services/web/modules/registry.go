package modules

import (
	"strings"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/authredirect"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/passkeys"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/shell"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Registry builds public/protected module sets from composition inputs.
type Registry struct{}

// BuildInput carries the dependencies and options needed to compose module sets.
type BuildInput struct {
	Dependencies              Dependencies
	Resolvers                 ModuleResolvers
	PublicOptions             PublicModuleOptions
	ProtectedOptions          ProtectedModuleOptions
	EnableExperimentalModules bool
}

// BuildOutput contains the composed module sets and derived health metadata.
type BuildOutput struct {
	Public    []Module
	Protected []Module
	Health    []dashboard.ServiceHealthEntry
}

// NewRegistry returns the default web module registry.
func NewRegistry() Registry {
	return Registry{}
}

// Build composes module sets for the requested stability mode.
func (Registry) Build(input BuildInput) BuildOutput {
	publicModules := DefaultPublicModules(input.Dependencies, input.Resolvers, input.PublicOptions)
	if input.EnableExperimentalModules {
		publicModules = append(publicModules, ExperimentalPublicModules()...)
	}

	protectedModules, health := buildProtectedModules(
		input.Dependencies,
		input.Resolvers,
		input.ProtectedOptions,
		input.EnableExperimentalModules,
	)

	return BuildOutput{
		Public:    publicModules,
		Protected: protectedModules,
		Health:    health,
	}
}

// DefaultPublicModules returns stable public web modules.
func DefaultPublicModules(deps Dependencies, res ModuleResolvers, opts PublicModuleOptions) []Module {
	authGateway := publicauth.NewGRPCAuthGateway(deps.AuthClient)
	return []Module{
		shell.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		passkeys.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		authredirect.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		discovery.New(),
		profile.NewWithGateway(profile.NewGRPCGateway(deps.SocialClient), deps.AssetBaseURL, res.ResolveSignedIn),
	}
}

// ExperimentalPublicModules returns opt-in public modules that are still scaffolded.
func ExperimentalPublicModules() []Module {
	return []Module{}
}

type PublicModuleOptions struct {
	RequestSchemePolicy requestmeta.SchemePolicy
}

// ProtectedModuleOptions controls variant behavior for protected module composition.
type ProtectedModuleOptions struct {
	// ChatFallbackPort is the derived chat service port passed to the campaigns module.
	ChatFallbackPort string

	// RequestSchemePolicy controls scheme resolution for scheme-sensitive behavior in protected modules.
	RequestSchemePolicy requestmeta.SchemePolicy
}

// DefaultProtectedModules returns stable authenticated web modules.
func DefaultProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []Module {
	modules, _ := buildProtectedModules(deps, res, opts, false)
	return modules
}

// ExperimentalProtectedModules returns protected modules when experimental campaigns are enabled.
func ExperimentalProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []Module {
	modules, _ := buildProtectedModules(deps, res, opts, true)
	return modules
}

func buildProtectedModules(
	deps Dependencies,
	res ModuleResolvers,
	opts ProtectedModuleOptions,
	experimentalCampaigns bool,
) ([]Module, []dashboard.ServiceHealthEntry) {
	base := modulehandler.NewBase(res.ResolveUserID, res.ResolveLanguage, res.ResolveViewer)
	campaignMod := NewStableCampaignModule(deps, base, opts.ChatFallbackPort)
	if experimentalCampaigns {
		campaignMod = NewExperimentalCampaignModule(deps, base, opts.ChatFallbackPort)
	}
	settingsMod := settings.New(settings.WithGateway(settings.NewGRPCGateway(deps.SocialClient, deps.AccountClient, deps.CredentialClient)), settings.WithBase(base), settings.WithSchemePolicy(opts.RequestSchemePolicy))
	notifMod := notifications.NewWithGateway(notifications.NewGRPCGateway(deps.NotificationClient), base)
	profileProbe := profile.NewWithGateway(profile.NewGRPCGateway(deps.SocialClient), deps.AssetBaseURL, res.ResolveSignedIn)

	// Dashboard's own health is derived from a probe module — the dashboard
	// module is constructed last because it receives the complete health list.
	dashGw := dashboard.NewGRPCGateway(deps.UserHubClient)
	dashProbe := dashboard.NewWithGateway(dashGw, base, nil)

	health := DeriveServiceHealth([]Module{
		profileProbe,
		settingsMod,
		notifMod,
		campaignMod,
		dashProbe,
	})

	dashMod := dashboard.NewWithGateway(dashGw, base, health)
	return []Module{dashMod, settingsMod, notifMod, campaignMod}, health
}

// DeriveServiceHealth builds health entries from modules that implement
// HealthReporter. Each module is the single source of truth for its own
// availability — new dependencies automatically affect health without
// manual registry updates.
func DeriveServiceHealth(modules []Module) []dashboard.ServiceHealthEntry {
	var entries []dashboard.ServiceHealthEntry
	for _, m := range modules {
		hr, ok := m.(module.HealthReporter)
		if !ok {
			continue
		}
		entries = append(entries, dashboard.ServiceHealthEntry{
			Label:     capitalizeLabel(m.ID()),
			Available: hr.Healthy(),
		})
	}
	return entries
}

func capitalizeLabel(id string) string {
	if id == "" {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

// defaultCampaignWorkflows returns the production workflow implementations
// keyed by their system label (lowercase).
func defaultCampaignWorkflows() map[string]campaigns.CharacterCreationWorkflow {
	return map[string]campaigns.CharacterCreationWorkflow{
		"daggerheart": daggerheart.New(),
	}
}

// NewStableCampaignModule returns a campaigns module configured for stable routes.
func NewStableCampaignModule(deps Dependencies, base modulehandler.Base, chatFallbackPort string) Module {
	return campaigns.NewStableWithGateway(newCampaignGateway(deps), base, chatFallbackPort, defaultCampaignWorkflows())
}

// NewExperimentalCampaignModule returns a campaigns module configured for experimental routes.
func NewExperimentalCampaignModule(deps Dependencies, base modulehandler.Base, chatFallbackPort string) Module {
	return campaigns.NewExperimentalWithGateway(newCampaignGateway(deps), base, chatFallbackPort, defaultCampaignWorkflows())
}

func newCampaignGateway(deps Dependencies) campaigns.CampaignGateway {
	return campaigns.NewGRPCGateway(campaigns.GRPCGatewayDeps{
		CampaignClient:           deps.CampaignClient,
		ParticipantClient:        deps.ParticipantClient,
		CharacterClient:          deps.CharacterClient,
		DaggerheartContentClient: deps.DaggerheartContentClient,
		SessionClient:            deps.SessionClient,
		InviteClient:             deps.InviteClient,
		AuthorizationClient:      deps.AuthorizationClient,
		AssetBaseURL:             deps.AssetBaseURL,
	})
}
