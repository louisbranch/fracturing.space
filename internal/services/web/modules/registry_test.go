package modules

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

var (
	emptyBase   modulehandler.Base
	emptyPolicy requestmeta.SchemePolicy
)

func TestDefaultModulesIncludeOnlyStableAreas(t *testing.T) {
	t.Parallel()

	public := DefaultPublicModules(Dependencies{}, ModuleResolvers{}, PublicModuleOptions{})
	protected := DefaultProtectedModules(Dependencies{}, ModuleResolvers{}, ProtectedModuleOptions{})
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

func TestExperimentalModulesAvailableWhenEnabled(t *testing.T) {
	t.Parallel()

	public := ExperimentalPublicModules()
	protected := ExperimentalProtectedModules(Dependencies{}, ModuleResolvers{}, ProtectedModuleOptions{})
	if len(public) != 0 {
		t.Fatalf("experimental public module count = %d, want %d", len(public), 0)
	}
	if len(protected) != 4 {
		t.Fatalf("experimental protected module count = %d, want %d", len(protected), 4)
	}
}

func TestStableModulesHaveUniquePrefixes(t *testing.T) {
	t.Parallel()

	public := DefaultPublicModules(Dependencies{}, ModuleResolvers{}, PublicModuleOptions{})
	protected := DefaultProtectedModules(Dependencies{}, ModuleResolvers{}, ProtectedModuleOptions{})
	seen := map[string]struct{}{}
	for _, module := range append(public, protected...) {
		mount, err := module.Mount()
		if err != nil {
			t.Fatalf("module %q mount error = %v", module.ID(), err)
		}
		if mount.Prefix == "" {
			t.Fatalf("module %q prefix is empty", module.ID())
		}
		if _, ok := seen[mount.Prefix]; ok {
			t.Fatalf("stable module duplicate mount prefix %q", mount.Prefix)
		}
		seen[mount.Prefix] = struct{}{}
	}
}

func TestExperimentalModulesHaveUniquePrefixes(t *testing.T) {
	t.Parallel()

	protected := ExperimentalProtectedModules(Dependencies{}, ModuleResolvers{}, ProtectedModuleOptions{})
	seen := map[string]struct{}{}
	for _, module := range protected {
		mount, err := module.Mount()
		if err != nil {
			t.Fatalf("module %q mount error = %v", module.ID(), err)
		}
		if mount.Prefix == "" {
			t.Fatalf("module %q prefix is empty", module.ID())
		}
		if _, ok := seen[mount.Prefix]; ok {
			t.Fatalf("experimental protected module duplicate mount prefix %q", mount.Prefix)
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
		CampaignClient:      stubCampaignClient{},
		ParticipantClient:   stubParticipantClient{},
		CharacterClient:     stubCharacterClient{},
		SessionClient:       stubSessionClient{},
		InviteClient:        stubInviteClient{},
		AuthorizationClient: stubAuthorizationClient{},
		UserHubClient:       stubUserHubClient{},
		SocialClient:        stubSocialClient{},
		AccountClient:       stubAccountClient{},
		CredentialClient:    stubCredentialClient{},
		NotificationClient:  stubNotificationClient{},
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
		// SocialClient nil → Profile degraded
		AccountClient:      stubAccountClient{},
		NotificationClient: stubNotificationClient{},
		// Settings missing SocialClient and CredentialClient → degraded
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

func TestRegistryBuildStableAndExperimental(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	stable := reg.Build(BuildInput{
		Dependencies:     Dependencies{},
		Resolvers:        ModuleResolvers{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	if len(stable.Public) != 5 {
		t.Fatalf("stable public module count = %d, want 5", len(stable.Public))
	}
	if len(stable.Protected) != 4 {
		t.Fatalf("stable protected module count = %d, want 4", len(stable.Protected))
	}
	if len(stable.Health) != 5 {
		t.Fatalf("stable health entry count = %d, want 5", len(stable.Health))
	}

	experimental := reg.Build(BuildInput{
		Dependencies:              Dependencies{},
		Resolvers:                 ModuleResolvers{},
		PublicOptions:             PublicModuleOptions{},
		ProtectedOptions:          ProtectedModuleOptions{},
		EnableExperimentalModules: true,
	})
	if len(experimental.Public) != 5 {
		t.Fatalf("experimental public module count = %d, want 5", len(experimental.Public))
	}
	if len(experimental.Protected) != 4 {
		t.Fatalf("experimental protected module count = %d, want 4", len(experimental.Protected))
	}
	if len(experimental.Health) != 5 {
		t.Fatalf("experimental health entry count = %d, want 5", len(experimental.Health))
	}
}

func TestRegistryBuildMatchesCompatibilityWrappers(t *testing.T) {
	t.Parallel()

	deps := Dependencies{}
	res := ModuleResolvers{}
	publicOpts := PublicModuleOptions{}
	protectedOpts := ProtectedModuleOptions{}

	reg := NewRegistry()
	built := reg.Build(BuildInput{
		Dependencies:     deps,
		Resolvers:        res,
		PublicOptions:    publicOpts,
		ProtectedOptions: protectedOpts,
	})

	wrappedPublic := DefaultPublicModules(deps, res, publicOpts)
	wrappedProtected := DefaultProtectedModules(deps, res, protectedOpts)

	assertModuleIDsEqual(t, built.Public, wrappedPublic, "public")
	assertModuleIDsEqual(t, built.Protected, wrappedProtected, "protected")
}

func assertModuleIDsEqual(t *testing.T, got []Module, want []Module, label string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s module count = %d, want %d", label, len(got), len(want))
	}
	for i := range got {
		if got[i].ID() != want[i].ID() {
			t.Fatalf("%s module[%d] id = %q, want %q", label, i, got[i].ID(), want[i].ID())
		}
	}
}

// buildHealthModules constructs modules from deps for health derivation tests.
func buildHealthModules(deps Dependencies) []Module {
	return []Module{
		campaigns.NewStableWithGateway(newCampaignGateway(deps), emptyBase, "", nil),
		dashboard.NewWithGateway(dashboard.NewGRPCGateway(deps.UserHubClient), emptyBase, nil),
		profile.NewWithGateway(profile.NewGRPCGateway(deps.SocialClient), "", nil),
		settings.New(settings.WithGateway(settings.NewGRPCGateway(deps.SocialClient, deps.AccountClient, deps.CredentialClient)), settings.WithBase(emptyBase), settings.WithSchemePolicy(emptyPolicy)),
		notifications.NewWithGateway(notifications.NewGRPCGateway(deps.NotificationClient), emptyBase),
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
