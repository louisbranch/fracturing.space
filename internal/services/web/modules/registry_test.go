package modules

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

var (
	emptyBase   modulehandler.Base
	emptyPolicy requestmeta.SchemePolicy
)

func TestDefaultModulesIncludeOnlyStableAreas(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	built := reg.Build(BuildInput{
		Dependencies:     Dependencies{},
		Resolvers:        ModuleResolvers{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	public := built.Public
	protected := built.Protected
	if len(public) != 5 {
		t.Fatalf("public module count = %d, want %d", len(public), 5)
	}
	if len(protected) != 4 {
		t.Fatalf("protected module count = %d, want %d", len(protected), 4)
	}

	if got := public[0].ID(); got != "public" {
		t.Fatalf("default public module id = %q, want %q", got, "public")
	}
	if got := public[1].ID(); got != "public-passkeys" {
		t.Fatalf("default public module[1] id = %q, want %q", got, "public-passkeys")
	}
	if got := public[2].ID(); got != "public-auth-redirect" {
		t.Fatalf("default public module[2] id = %q, want %q", got, "public-auth-redirect")
	}
	if got := public[3].ID(); got != "discovery" {
		t.Fatalf("default public module[3] id = %q, want %q", got, "discovery")
	}
	if got := public[4].ID(); got != "profile" {
		t.Fatalf("default public module[4] id = %q, want %q", got, "profile")
	}
	if got := protected[0].ID(); got != "dashboard" {
		t.Fatalf("default protected module[0] id = %q, want %q", got, "dashboard")
	}
	if got := protected[1].ID(); got != "settings" {
		t.Fatalf("default protected module[1] id = %q, want %q", got, "settings")
	}
	if got := protected[2].ID(); got != "notifications" {
		t.Fatalf("default protected module[2] id = %q, want %q", got, "notifications")
	}
	if got := protected[3].ID(); got != "campaigns" {
		t.Fatalf("default protected module[3] id = %q, want %q", got, "campaigns")
	}
}

func TestDefaultProtectedModulesDelegatesToBuilder(t *testing.T) {
	t.Parallel()

	deps := Dependencies{}
	resolvers := ModuleResolvers{}
	opts := ProtectedModuleOptions{}

	modules := defaultProtectedModules(deps, resolvers, opts)
	builtModules, _ := buildProtectedModules(deps, resolvers, opts)
	if len(modules) != len(builtModules) {
		t.Fatalf("defaultProtectedModules len = %d, want %d", len(modules), len(builtModules))
	}
	for i := range modules {
		if modules[i].ID() != builtModules[i].ID() {
			t.Fatalf("module[%d].ID() = %q, want %q", i, modules[i].ID(), builtModules[i].ID())
		}
	}
}

func TestModulesHaveUniquePrefixes(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	built := reg.Build(BuildInput{
		Dependencies:     Dependencies{},
		Resolvers:        ModuleResolvers{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	seen := map[string]struct{}{}
	for _, module := range append(built.Public, built.Protected...) {
		mount, err := module.Mount()
		if err != nil {
			t.Fatalf("module %q mount error = %v", module.ID(), err)
		}
		if mount.Prefix == "" {
			t.Fatalf("module %q prefix is empty", module.ID())
		}
		if _, ok := seen[mount.Prefix]; ok {
			t.Fatalf("duplicate module mount prefix %q", mount.Prefix)
		}
		seen[mount.Prefix] = struct{}{}
	}
}

func TestDeriveServiceHealthAllNilDeps(t *testing.T) {
	t.Parallel()

	modules := buildHealthModules(Dependencies{})
	health := DeriveServiceHealth(modules)
	if len(health) != 5 {
		t.Fatalf("health entry count = %d, want 5", len(health))
	}
	for _, e := range health {
		if e.Available {
			t.Fatalf("entry %q Available = true, want false with nil deps", e.Label)
		}
	}
}

func TestDeriveServiceHealthAllPresent(t *testing.T) {
	t.Parallel()

	deps := Dependencies{
		CampaignClient:       stubCampaignClient{},
		ParticipantClient:    stubParticipantClient{},
		CharacterClient:      stubCharacterClient{},
		SessionClient:        stubSessionClient{},
		InviteClient:         stubInviteClient{},
		AuthorizationClient:  stubAuthorizationClient{},
		UserHubClient:        stubUserHubClient{},
		ProfileSocialClient:  stubSocialClient{},
		SettingsSocialClient: stubSocialClient{},
		AccountClient:        stubAccountClient{},
		CredentialClient:     stubCredentialClient{},
		NotificationClient:   stubNotificationClient{},
	}
	modules := buildHealthModules(deps)
	health := DeriveServiceHealth(modules)
	if len(health) != 5 {
		t.Fatalf("health entry count = %d, want 5", len(health))
	}
	for _, e := range health {
		if !e.Available {
			t.Fatalf("entry %q Available = false, want true with all deps present", e.Label)
		}
	}
}

func TestDeriveServiceHealthMixedDeps(t *testing.T) {
	t.Parallel()

	deps := Dependencies{
		CampaignClient: stubCampaignClient{},
		// Campaigns is missing ParticipantClient etc → degraded
		// UserHubClient nil → Dashboard degraded
		// ProfileSocialClient nil → Profile degraded
		AccountClient:      stubAccountClient{},
		NotificationClient: stubNotificationClient{},
		// Settings missing SettingsSocialClient and CredentialClient → degraded
	}
	modules := buildHealthModules(deps)
	health := DeriveServiceHealth(modules)

	want := map[string]bool{
		"Campaigns":     false, // partial game clients
		"Dashboard":     false,
		"Profile":       false,
		"Settings":      false, // partial settings clients
		"Notifications": true,
	}
	for _, e := range health {
		expected, ok := want[e.Label]
		if !ok {
			t.Fatalf("unexpected health entry label %q", e.Label)
		}
		if e.Available != expected {
			t.Fatalf("entry %q Available = %v, want %v", e.Label, e.Available, expected)
		}
	}
}

func TestModuleHealthyReturnsFalseForUnavailableGateway(t *testing.T) {
	t.Parallel()

	m := campaigns.New()
	if m.Healthy() {
		t.Fatalf("campaigns.New() Healthy = true, want false for zero-value module")
	}

	dm := dashboard.New()
	if dm.Healthy() {
		t.Fatalf("dashboard.New() Healthy = true, want false for zero-value module")
	}

	sm := settings.New()
	if sm.Healthy() {
		t.Fatalf("settings.New() Healthy = true, want false for zero-value module")
	}

	nm := notifications.New()
	if nm.Healthy() {
		t.Fatalf("notifications.New() Healthy = true, want false for zero-value module")
	}
}

func TestRegistryBuildComposesExpectedModules(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	built := reg.Build(BuildInput{
		Dependencies:     Dependencies{},
		Resolvers:        ModuleResolvers{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	if len(built.Public) != 5 {
		t.Fatalf("public module count = %d, want 5", len(built.Public))
	}
	if len(built.Protected) != 4 {
		t.Fatalf("protected module count = %d, want 4", len(built.Protected))
	}
	if len(built.Health) != 5 {
		t.Fatalf("health entry count = %d, want 5", len(built.Health))
	}
}

// buildHealthModules constructs modules from deps for health derivation tests.
func buildHealthModules(deps Dependencies) []Module {
	return []Module{
		campaigns.NewStableWithGateway(newCampaignGateway(deps), emptyBase, "", nil),
		dashboard.NewWithGateway(dashboardgateway.NewGRPCGateway(deps.UserHubClient), emptyBase, nil),
		profile.NewWithGateway(profilegateway.NewGRPCGateway(deps.ProfileSocialClient), "", nil),
		settings.New(settings.WithGateway(settingsgateway.NewGRPCGateway(deps.SettingsSocialClient, deps.AccountClient, deps.CredentialClient)), settings.WithBase(emptyBase), settings.WithSchemePolicy(emptyPolicy)),
		notifications.NewWithGateway(notificationsgateway.NewGRPCGateway(deps.NotificationClient), emptyBase),
	}
}

// Stubs satisfy the client interfaces with embedded interface types.
// Methods are never called — only nil-checks matter for DeriveServiceHealth.
type stubCampaignClient struct{ campaigns.CampaignClient }
type stubParticipantClient struct{ campaigns.ParticipantClient }
type stubCharacterClient struct{ campaigns.CharacterClient }
type stubSessionClient struct{ campaigns.SessionClient }
type stubInviteClient struct{ campaigns.InviteClient }
type stubAuthorizationClient struct{ campaigns.AuthorizationClient }
type stubUserHubClient struct{ dashboard.UserHubClient }
type stubSocialClient struct{ settings.SocialClient }
type stubAccountClient struct{ settings.AccountClient }
type stubCredentialClient struct{ settings.CredentialClient }
type stubNotificationClient struct {
	notifications.NotificationClient
}
