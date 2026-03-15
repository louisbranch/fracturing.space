package web

import (
	"errors"
	"fmt"
	"strings"

	grpc "google.golang.org/grpc"
)

// DependencyBinder maps one connected backend dependency into the assembled
// web runtime bundle.
type DependencyBinder func(*DependencyBundle, *grpc.ClientConn)

// DependencyValidator reports whether one dependency contributed the required
// runtime clients into the assembled web dependency bundle.
type DependencyValidator func(DependencyBundle) error

// StartupDependencyPolicy describes whether one backend dependency is required
// for production web startup or may degrade owned surfaces when unavailable.
type StartupDependencyPolicy string

const (
	// StartupDependencyRequired blocks production startup when unavailable.
	StartupDependencyRequired StartupDependencyPolicy = "required"
	// StartupDependencyOptional degrades only owned surfaces when unavailable.
	StartupDependencyOptional StartupDependencyPolicy = "optional"
)

// StartupDependencyDescriptor describes one backend dependency the web service
// can bind during startup. Command-layer startup policy consumes this table
// instead of owning binder selection itself.
type StartupDependencyDescriptor struct {
	Name       string
	Policy     StartupDependencyPolicy
	Capability string
	Surfaces   []string
	Bind       DependencyBinder
	Validate   DependencyValidator
}

const (
	// DependencyNameAuth identifies the auth backend dependency.
	DependencyNameAuth = "auth"
	// DependencyNameSocial identifies the social backend dependency.
	DependencyNameSocial = "social"
	// DependencyNameGame identifies the game backend dependency.
	DependencyNameGame = "game"
	// DependencyNameAI identifies the AI backend dependency.
	DependencyNameAI = "ai"
	// DependencyNameDiscovery identifies the discovery backend dependency.
	DependencyNameDiscovery = "discovery"
	// DependencyNameUserHub identifies the userhub backend dependency.
	DependencyNameUserHub = "userhub"
	// DependencyNameNotifications identifies the notifications backend dependency.
	DependencyNameNotifications = "notifications"
	// DependencyNameStatus identifies the status backend dependency.
	DependencyNameStatus = "status"
)

var startupDependencyDescriptors = []StartupDependencyDescriptor{
	{
		Name:       DependencyNameAuth,
		Policy:     StartupDependencyRequired,
		Capability: "web.auth.integration",
		Surfaces:   []string{"principal", "publicauth", "profile", "settings"},
		Bind:       BindAuthDependency,
		Validate: func(bundle DependencyBundle) error {
			return requireDependencyFields(DependencyNameAuth,
				fieldCheck{name: "principal.session", configured: bundle.Principal.SessionClient != nil},
				fieldCheck{name: "principal.account", configured: bundle.Principal.AccountClient != nil},
				fieldCheck{name: "modules.publicauth.auth", configured: bundle.Modules.PublicAuth.AuthClient != nil},
				fieldCheck{name: "modules.profile.auth", configured: bundle.Modules.Profile.AuthClient != nil},
				fieldCheck{name: "modules.settings.account", configured: bundle.Modules.Settings.AccountClient != nil},
				fieldCheck{name: "modules.settings.passkey", configured: bundle.Modules.Settings.PasskeyClient != nil},
				fieldCheck{name: "modules.campaigns.auth", configured: bundle.Modules.Campaigns.AuthClient != nil},
				fieldCheck{name: "modules.invite.auth", configured: bundle.Modules.Invite.AuthClient != nil},
			)
		},
	},
	{
		Name:       DependencyNameSocial,
		Policy:     StartupDependencyRequired,
		Capability: "web.social.integration",
		Surfaces:   []string{"principal", "profile", "settings", "campaigns"},
		Bind:       BindSocialDependency,
		Validate: func(bundle DependencyBundle) error {
			return requireDependencyFields(DependencyNameSocial,
				fieldCheck{name: "principal.social", configured: bundle.Principal.SocialClient != nil},
				fieldCheck{name: "modules.profile.social", configured: bundle.Modules.Profile.SocialClient != nil},
				fieldCheck{name: "modules.settings.social", configured: bundle.Modules.Settings.SocialClient != nil},
				fieldCheck{name: "modules.campaigns.social", configured: bundle.Modules.Campaigns.SocialClient != nil},
			)
		},
	},
	{
		Name:       DependencyNameGame,
		Policy:     StartupDependencyRequired,
		Capability: "web.game.integration",
		Surfaces:   []string{"campaigns", "dashboard-sync"},
		Bind:       BindGameDependency,
		Validate: func(bundle DependencyBundle) error {
			return requireDependencyFields(DependencyNameGame,
				fieldCheck{name: "modules.campaigns.campaign", configured: bundle.Modules.Campaigns.CampaignClient != nil},
				fieldCheck{name: "modules.campaigns.participant", configured: bundle.Modules.Campaigns.ParticipantClient != nil},
				fieldCheck{name: "modules.campaigns.character", configured: bundle.Modules.Campaigns.CharacterClient != nil},
				fieldCheck{name: "modules.campaigns.session", configured: bundle.Modules.Campaigns.SessionClient != nil},
				fieldCheck{name: "modules.campaigns.invite", configured: bundle.Modules.Campaigns.InviteClient != nil},
				fieldCheck{name: "modules.campaigns.authorization", configured: bundle.Modules.Campaigns.AuthorizationClient != nil},
				fieldCheck{name: "modules.campaigns.daggerheart-content", configured: bundle.Modules.Campaigns.DaggerheartContentClient != nil},
				fieldCheck{name: "modules.campaigns.daggerheart-asset", configured: bundle.Modules.Campaigns.DaggerheartAssetClient != nil},
				fieldCheck{name: "modules.campaigns.fork", configured: bundle.Modules.Campaigns.ForkClient != nil},
				fieldCheck{name: "modules.invite.invite", configured: bundle.Modules.Invite.InviteClient != nil},
				fieldCheck{name: "modules.dashboard-sync.game-events", configured: bundle.Modules.DashboardSync.GameEventClient != nil},
			)
		},
	},
	{
		Name:       DependencyNameAI,
		Policy:     StartupDependencyOptional,
		Capability: "web.ai.integration",
		Surfaces:   []string{"settings.ai", "campaigns.ai"},
		Bind:       BindAIDependency,
	},
	{
		Name:       DependencyNameDiscovery,
		Policy:     StartupDependencyOptional,
		Capability: "web.discovery.integration",
		Surfaces:   []string{"discovery"},
		Bind:       BindDiscoveryDependency,
	},
	{
		Name:       DependencyNameUserHub,
		Policy:     StartupDependencyOptional,
		Capability: "web.userhub.integration",
		Surfaces:   []string{"dashboard", "dashboard-sync"},
		Bind:       BindUserHubDependency,
	},
	{
		Name:       DependencyNameNotifications,
		Policy:     StartupDependencyOptional,
		Capability: "web.notifications.integration",
		Surfaces:   []string{"principal", "notifications"},
		Bind:       BindNotificationsDependency,
	},
	{
		Name:       DependencyNameStatus,
		Policy:     StartupDependencyOptional,
		Capability: "web.status.integration",
		Surfaces:   []string{"dashboard.health"},
		Bind:       BindStatusDependency,
	},
}

// StartupDependencyDescriptors returns the stable startup dependency descriptor
// table consumed by command-layer bootstrap policy.
func StartupDependencyDescriptors() []StartupDependencyDescriptor {
	descriptors := make([]StartupDependencyDescriptor, len(startupDependencyDescriptors))
	for i, descriptor := range startupDependencyDescriptors {
		descriptor.Surfaces = append([]string(nil), descriptor.Surfaces...)
		descriptors[i] = descriptor
	}
	return descriptors
}

// LookupStartupDependencyDescriptor returns the service-owned descriptor for
// one backend dependency.
func LookupStartupDependencyDescriptor(name string) (StartupDependencyDescriptor, bool) {
	for _, descriptor := range startupDependencyDescriptors {
		if descriptor.Name == name {
			descriptor.Surfaces = append([]string(nil), descriptor.Surfaces...)
			return descriptor, true
		}
	}
	return StartupDependencyDescriptor{}, false
}

// validateRequiredDependencyBundle enforces the fail-closed production startup
// contract before the root handler is composed.
func validateRequiredDependencyBundle(bundle *DependencyBundle) error {
	if bundle == nil {
		return errors.New("web dependencies are required")
	}
	for _, descriptor := range startupDependencyDescriptors {
		if descriptor.Policy != StartupDependencyRequired || descriptor.Validate == nil {
			continue
		}
		if err := descriptor.Validate(*bundle); err != nil {
			return err
		}
	}
	return nil
}

// fieldCheck records one required runtime client that a dependency binder must
// contribute to the assembled web bundle.
type fieldCheck struct {
	name       string
	configured bool
}

// requireDependencyFields converts missing required runtime clients into one
// dependency-scoped startup error message.
func requireDependencyFields(dependencyName string, fields ...fieldCheck) error {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.configured {
			continue
		}
		name := strings.TrimSpace(field.name)
		if name != "" {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"required dependency %q is incomplete: missing %s",
		strings.TrimSpace(dependencyName),
		strings.Join(missing, ", "),
	)
}
