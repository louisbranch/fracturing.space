package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpcHealth "google.golang.org/grpc/health"
	grpcHealthV1 "google.golang.org/grpc/health/grpc_health_v1"
)

func testManagedConnFactory(t *testing.T) managedConnFactory {
	t.Helper()
	return func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		cfg.StatusReporter = nil
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
}

func mustDependencyRequirements(t *testing.T, cfg Config) []dependencyRequirement {
	t.Helper()

	requirements, err := dependencyRequirements(cfg, nil)
	if err != nil {
		t.Fatalf("dependencyRequirements() error = %v", err)
	}
	return requirements
}

func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "localhost:8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "localhost:8080")
	}
	if cfg.GameAddr != "game:8082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "game:8082")
	}
	if cfg.PlayHTTPAddr != "localhost:8094" {
		t.Fatalf("PlayHTTPAddr = %q, want %q", cfg.PlayHTTPAddr, "localhost:8094")
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("AuthAddr = %q, want %q", cfg.AuthAddr, "auth:8083")
	}
	if cfg.SocialAddr != "social:8090" {
		t.Fatalf("SocialAddr = %q, want %q", cfg.SocialAddr, "social:8090")
	}
	if cfg.AIAddr != "ai:8087" {
		t.Fatalf("AIAddr = %q, want %q", cfg.AIAddr, "ai:8087")
	}
	if cfg.NotificationsAddr != "notifications:8088" {
		t.Fatalf("NotificationsAddr = %q, want %q", cfg.NotificationsAddr, "notifications:8088")
	}
	if cfg.UserHubAddr != "userhub:8092" {
		t.Fatalf("UserHubAddr = %q, want %q", cfg.UserHubAddr, "userhub:8092")
	}
	if cfg.StatusAddr != "status:8093" {
		t.Fatalf("StatusAddr = %q, want %q", cfg.StatusAddr, "status:8093")
	}
	if cfg.InviteAddr != "invite:8095" {
		t.Fatalf("InviteAddr = %q, want %q", cfg.InviteAddr, "invite:8095")
	}
	if cfg.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = %t, want false", cfg.TrustForwardedProto)
	}
	if cfg.AssetBaseURL != "" {
		t.Fatalf("AssetBaseURL = %q, want empty", cfg.AssetBaseURL)
	}
}

func TestParseConfigOverrideHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-http-addr", "127.0.0.1:9002"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9002" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9002")
	}
}

func TestParseConfigOverrideGameAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-game-addr", "127.0.0.1:19082"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.GameAddr != "127.0.0.1:19082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "127.0.0.1:19082")
	}
}

func TestParseConfigOverridePlayHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-play-http-addr", "127.0.0.1:18094"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.PlayHTTPAddr != "127.0.0.1:18094" {
		t.Fatalf("PlayHTTPAddr = %q, want %q", cfg.PlayHTTPAddr, "127.0.0.1:18094")
	}
}

func TestParseConfigOverrideTrustForwardedProto(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-trust-forwarded-proto"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = %t, want true", cfg.TrustForwardedProto)
	}
}

func TestParseConfigOverrideDependencyAddrFlags(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	args := make([]string, 0, len(dependencyAddressBindingNames())*2)
	want := make(map[string]string, len(dependencyAddressBindingNames()))

	for i, name := range dependencyAddressBindingNames() {
		value := fmt.Sprintf("127.0.0.%d:%d", 1+i, 20000+i)
		args = append(args, "-"+dependencyAddressFlagName(name), value)
		want[name] = value
	}

	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	for _, name := range dependencyAddressBindingNames() {
		binding, ok := dependencyAddressBindingForName(name)
		if !ok {
			t.Fatalf("missing dependency address binding for %q", name)
		}
		got := strings.TrimSpace(*binding.address(&cfg))
		if got != want[name] {
			t.Fatalf("dependency address %q = %q, want %q", name, got, want[name])
		}
	}
}

func TestParseConfigOverrideUserHubAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-userhub-addr", "127.0.0.1:18092"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.UserHubAddr != "127.0.0.1:18092" {
		t.Fatalf("UserHubAddr = %q, want %q", cfg.UserHubAddr, "127.0.0.1:18092")
	}
}

func TestParseConfigOverrideNotificationsAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-notifications-addr", "127.0.0.1:18088"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.NotificationsAddr != "127.0.0.1:18088" {
		t.Fatalf("NotificationsAddr = %q, want %q", cfg.NotificationsAddr, "127.0.0.1:18088")
	}
}

func TestParseConfigOverrideStatusAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-status-addr", "127.0.0.1:18093"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.StatusAddr != "127.0.0.1:18093" {
		t.Fatalf("StatusAddr = %q, want %q", cfg.StatusAddr, "127.0.0.1:18093")
	}
}

func testDependencyConfig() Config {
	return Config{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		GameAddr:          "game:8082",
		InviteAddr:        "invite:8095",
		AIAddr:            "ai:8087",
		DiscoveryAddr:     "discovery:8091",
		UserHubAddr:       "userhub:8092",
		NotificationsAddr: "notifications:8088",
		StatusAddr:        "status:8093",
	}
}

func copyDependencyAddressResolvers() map[string]dependencyAddressResolver {
	return dependencyAddressResolverDefaults()
}

func TestBootstrapRuntimeDependenciesReturnsMissingAddressError(t *testing.T) {
	cfg := testDependencyConfig()
	cfg.AuthAddr = "   "
	cfg.SocialAddr = " "
	cfg.GameAddr = ""

	_, err := bootstrapRuntimeDependencies(
		context.Background(),
		cfg,
		platformstatus.NewReporter("web", nil),
		&bootstrapOptions{NewConn: testManagedConnFactory(t)},
	)
	if err == nil {
		t.Fatal("expected missing required address error")
	}

	var missing MissingRequiredStartupDependencyAddressesError
	if !errors.As(err, &missing) {
		t.Fatalf("bootstrapRuntimeDependencies() error type = %T, want MissingRequiredStartupDependencyAddressesError", err)
	}

	want := []string{web.DependencyNameAuth, web.DependencyNameGame, web.DependencyNameSocial}
	if !slices.Equal(missing.Missing, want) {
		t.Fatalf("missing required addresses = %#v, want %#v", missing.Missing, want)
	}
}

func TestBootstrapRuntimeDependenciesReturnsResolverContractErrorOnCoverageDrift(t *testing.T) {
	drifted := copyDependencyAddressResolvers()
	delete(drifted, web.DependencyNameSocial)

	_, err := bootstrapRuntimeDependencies(
		context.Background(),
		testDependencyConfig(),
		platformstatus.NewReporter("web", nil),
		&bootstrapOptions{NewConn: testManagedConnFactory(t), Resolvers: drifted},
	)
	if err == nil {
		t.Fatal("expected dependency resolver contract mismatch error")
	}

	var contractErr DependencyAddressResolverContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("bootstrapRuntimeDependencies() error type = %T, want DependencyAddressResolverContractError", err)
	}
	if !slices.Equal(contractErr.Missing, []string{web.DependencyNameSocial}) {
		t.Fatalf("resolver contract mismatch = %#v, want missing [%q]", contractErr, web.DependencyNameSocial)
	}
}

func TestDependencyRequirementsRequiredPolicy(t *testing.T) {
	t.Parallel()

	requirements := mustDependencyRequirements(t, testDependencyConfig())
	requiredNames := make([]string, 0, len(requirements))
	for _, dep := range requirements {
		if dep.policy == web.StartupDependencyRequired {
			requiredNames = append(requiredNames, dep.name)
		}
	}
	slices.Sort(requiredNames)
	want := []string{web.DependencyNameAuth, web.DependencyNameGame, web.DependencyNameInvite, web.DependencyNameSocial}
	if !slices.Equal(requiredNames, want) {
		t.Fatalf("required dependencies = %v, want %v", requiredNames, want)
	}
}

func TestDependencyRequirementsRejectMissingRequiredAddress(t *testing.T) {
	t.Parallel()

	cfg := testDependencyConfig()
	cfg.AuthAddr = "   "
	_, err := dependencyRequirements(cfg, nil)
	if err == nil {
		t.Fatal("expected required-address dependency error")
	}
	var errMissingAddress MissingRequiredStartupDependencyAddressesError
	if !errors.As(err, &errMissingAddress) {
		t.Fatalf("dependencyRequirements() error type = %T, want MissingRequiredStartupDependencyAddressesError", err)
	}
	if len(errMissingAddress.Missing) != 1 || errMissingAddress.Missing[0] != web.DependencyNameAuth {
		t.Fatalf("dependencyRequirements() missing addresses = %#v, want [%q]", errMissingAddress.Missing, web.DependencyNameAuth)
	}
}

func TestDependencyAddressBindingsCoverStartupDescriptors(t *testing.T) {
	t.Parallel()

	descriptors := web.StartupDependencyDescriptors()
	descriptorNames := make(map[string]struct{}, len(descriptors))
	missing := make([]string, 0)
	for _, descriptor := range descriptors {
		name := strings.TrimSpace(descriptor.Name)
		if name == "" {
			continue
		}
		descriptorNames[name] = struct{}{}
		if _, ok := dependencyAddressBindingForName(name); !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("dependency address bindings missing for %v", missing)
	}

	extras := make([]string, 0)
	for _, name := range dependencyAddressBindingNames() {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := descriptorNames[name]; !ok {
			extras = append(extras, name)
		}
	}
	if len(extras) > 0 {
		t.Fatalf("dependency address bindings include unknown dependencies %v", extras)
	}
}

func TestDependencyRequirementsCollectsAllMissingRequiredAddresses(t *testing.T) {
	t.Parallel()

	cfg := testDependencyConfig()
	cfg.AuthAddr = "   "
	cfg.GameAddr = "   "
	cfg.InviteAddr = ""
	cfg.SocialAddr = ""
	_, err := dependencyRequirements(cfg, nil)
	if err == nil {
		t.Fatal("expected required-address dependency error")
	}
	var errMissingAddress MissingRequiredStartupDependencyAddressesError
	if !errors.As(err, &errMissingAddress) {
		t.Fatalf("dependencyRequirements() error type = %T, want MissingRequiredStartupDependencyAddressesError", err)
	}
	want := []string{web.DependencyNameAuth, web.DependencyNameGame, web.DependencyNameInvite, web.DependencyNameSocial}
	if !slices.Equal(errMissingAddress.Missing, want) {
		// Ordered by canonical sorted error payload.
		t.Fatalf("dependencyRequirements() missing addresses = %#v, want %#v", errMissingAddress.Missing, want)
	}
}

func TestDependencyAddressResolverForNameRejectsUnknownDependency(t *testing.T) {
	t.Parallel()

	if got := dependencyAddressResolverForName("ghost-dependency"); got != nil {
		t.Fatalf("dependencyAddressResolverForName(ghost) = %#v, want nil", got)
	}
}

func TestDependencyRequirementsWithResolversRejectsCoverageDrift(t *testing.T) {
	t.Parallel()

	_, err := dependencyRequirementsWithResolvers(testDependencyConfig(), nil, nil)
	if err == nil {
		t.Fatal("expected resolver contract mismatch for nil resolver map")
	}
	var contractErr DependencyAddressResolverContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("dependencyRequirementsWithResolvers() error type = %T, want DependencyAddressResolverContractError", err)
	}
	want := []string{
		web.DependencyNameAI,
		web.DependencyNameAuth,
		web.DependencyNameDiscovery,
		web.DependencyNameGame,
		web.DependencyNameInvite,
		web.DependencyNameNotifications,
		web.DependencyNameSocial,
		web.DependencyNameStatus,
		web.DependencyNameUserHub,
	}
	if !slices.Equal(contractErr.Missing, want) {
		t.Fatalf("dependency resolver contract mismatch missing = %#v, want %#v", contractErr.Missing, want)
	}
}

func TestDependencyAddressResolverDefaultsAreSnapshotSafe(t *testing.T) {
	t.Parallel()

	mutatedDefaults := copyDependencyAddressResolvers()
	delete(mutatedDefaults, web.DependencyNameGame)

	_, err := dependencyRequirementsWithResolvers(testDependencyConfig(), nil, mutatedDefaults)
	if err == nil {
		t.Fatal("expected resolver contract mismatch from mutated snapshot")
	}
	var contractErr DependencyAddressResolverContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("dependencyRequirementsWithResolvers() error type = %T, want DependencyAddressResolverContractError", err)
	}
	if !slices.Equal(contractErr.Missing, []string{web.DependencyNameGame}) {
		t.Fatalf("mutated resolver contract mismatch missing = %#v, want missing [%q]", contractErr, web.DependencyNameGame)
	}

	_, err = dependencyRequirements(testDependencyConfig(), nil)
	if err != nil {
		t.Fatalf("dependencyRequirements() should use stable defaults; got error = %v", err)
	}

	deps, err := bootstrapRuntimeDependencies(
		context.Background(),
		testDependencyConfig(),
		platformstatus.NewReporter("web", nil),
		&bootstrapOptions{NewConn: testManagedConnFactory(t)},
	)
	if err != nil {
		t.Fatalf("bootstrapRuntimeDependencies() should use stable defaults; got error = %v", err)
	}
	defer deps.close()
}

func TestDependencyRequirementsCoverAllServiceOwnedDescriptors(t *testing.T) {
	t.Parallel()

	requirements := mustDependencyRequirements(t, testDependencyConfig())
	descriptors := web.StartupDependencyDescriptors()
	if len(requirements) != len(descriptors) {
		t.Fatalf("dependency requirements = %d, want %d service descriptors", len(requirements), len(descriptors))
	}
}

func TestDependencyRequirementsOwnedSurfacesAreExplicit(t *testing.T) {
	t.Parallel()

	requirements := mustDependencyRequirements(t, testDependencyConfig())
	got := map[string][]string{}
	for _, dep := range requirements {
		if len(dep.surfaces) == 0 {
			t.Fatalf("dependency %q has no owned surfaces", dep.name)
		}
		got[dep.name] = dep.surfaces
	}

	tests := map[string][]string{
		web.DependencyNameAuth:          {"principal", "publicauth", "profile", "settings"},
		web.DependencyNameSocial:        {"principal", "profile", "settings", "campaigns"},
		web.DependencyNameGame:          {"campaigns", "dashboard-sync"},
		web.DependencyNameInvite:        {"campaigns", "invite"},
		web.DependencyNameAI:            {"settings.ai", "campaigns.ai"},
		web.DependencyNameDiscovery:     {"discovery"},
		web.DependencyNameUserHub:       {"dashboard", "dashboard-sync"},
		web.DependencyNameNotifications: {"principal", "notifications"},
		web.DependencyNameStatus:        {"dashboard.health"},
	}
	for name, want := range tests {
		if !slices.Equal(got[name], want) {
			t.Fatalf("dependency %q surfaces = %v, want %v", name, got[name], want)
		}
	}
}

func TestDependencyRequirementsAddressCoverageHasNoCoverageDrift(t *testing.T) {
	t.Parallel()

	if err := validateDependencyAddressResolversCoverage(); err != nil {
		t.Fatalf("validateDependencyAddressResolversCoverage() error = %v", err)
	}
}

func TestValidateDependencyAddressResolversCoverageReportsMissingResolver(t *testing.T) {
	t.Parallel()

	resolvers := make(map[string]dependencyAddressResolver, len(copyDependencyAddressResolvers()))
	for name, resolve := range dependencyAddressResolverDefaults() {
		resolvers[name] = resolve
	}
	delete(resolvers, web.DependencyNameSocial)

	err := validateDependencyAddressResolversCoverageWithResolvers(resolvers)
	if err == nil {
		t.Fatal("expected contract mismatch for missing social resolver")
	}
	var contractErr DependencyAddressResolverContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("validateDependencyAddressResolversCoverageWithResolvers() error type = %T, want DependencyAddressResolverContractError", err)
	}
	if len(contractErr.Missing) != 1 || contractErr.Missing[0] != web.DependencyNameSocial {
		t.Fatalf("dependency resolver contract mismatch = %#v, want missing=[%q]", contractErr, web.DependencyNameSocial)
	}
}

func TestValidateDependencyAddressResolversCoverageReportsExtraResolver(t *testing.T) {
	t.Parallel()

	resolvers := make(map[string]dependencyAddressResolver, len(copyDependencyAddressResolvers())+1)
	for name, resolve := range dependencyAddressResolverDefaults() {
		resolvers[name] = resolve
	}
	resolvers["ghost-service"] = func(cfg Config) string {
		return cfg.HTTPAddr
	}

	err := validateDependencyAddressResolversCoverageWithResolvers(resolvers)
	if err == nil {
		t.Fatal("expected contract mismatch for extra resolver")
	}
	var contractErr DependencyAddressResolverContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("validateDependencyAddressResolversCoverageWithResolvers() error type = %T, want DependencyAddressResolverContractError", err)
	}
	if len(contractErr.Extra) != 1 || contractErr.Extra[0] != "ghost-service" {
		t.Fatalf("dependency resolver contract mismatch = %#v, want extras=[%q]", contractErr, "ghost-service")
	}
}

func TestDependencyRequirementsCapabilitiesAreUnique(t *testing.T) {
	t.Parallel()

	requirements := mustDependencyRequirements(t, testDependencyConfig())
	seen := map[string]struct{}{}
	for _, dep := range requirements {
		if strings.TrimSpace(dep.capability) == "" {
			t.Fatalf("dependency %q has empty capability", dep.name)
		}
		if _, ok := seen[dep.capability]; ok {
			t.Fatalf("duplicate capability %q", dep.capability)
		}
		seen[dep.capability] = struct{}{}
	}
	if len(seen) != len(requirements) {
		t.Fatalf("unique capabilities = %d, want %d", len(seen), len(requirements))
	}
}

func TestDependencyRequirementManagedConnModeMatchesPolicy(t *testing.T) {
	t.Parallel()

	if got := managedConnMode(web.StartupDependencyRequired); got != platformgrpc.ModeRequired {
		t.Fatalf("required managedConnMode = %v, want %v", got, platformgrpc.ModeRequired)
	}
	if got := managedConnMode(web.StartupDependencyOptional); got != platformgrpc.ModeOptional {
		t.Fatalf("optional managedConnMode = %v, want %v", got, platformgrpc.ModeOptional)
	}
}

func TestBootstrapDependenciesWiresAllClients(t *testing.T) {
	cfg := testDependencyConfig()
	cfg.AssetBaseURL = "https://cdn.example.com/assets"
	requirements := mustDependencyRequirements(t, cfg)
	reporter := platformstatus.NewReporter("web", nil)

	bundle, conns, err := bootstrapDependencies(context.Background(), requirements, cfg.AssetBaseURL, reporter, nil, testManagedConnFactory(t))
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns, nil)

	if len(conns) != 9 {
		t.Fatalf("managed conns = %d, want 9", len(conns))
	}
	if bundle.Principal.SessionClient == nil {
		t.Fatal("expected principal session client")
	}
	if bundle.Modules.Campaigns.CampaignClient == nil {
		t.Fatal("expected campaign client")
	}
	if bundle.Modules.Campaigns.AuthClient == nil {
		t.Fatal("expected campaign auth client")
	}
	if bundle.Modules.Settings.CredentialClient == nil {
		t.Fatal("expected credential client")
	}
	if bundle.Modules.Profile.AuthClient == nil {
		t.Fatalf("expected profile auth client")
	}
	if bundle.Modules.Profile.SocialClient == nil {
		t.Fatal("expected profile social client")
	}
	if bundle.Modules.Settings.SocialClient == nil {
		t.Fatal("expected settings social client")
	}
	if bundle.Modules.Dashboard.UserHubClient == nil {
		t.Fatal("expected userhub client")
	}
	if bundle.Modules.Dashboard.StatusClient == nil {
		t.Fatal("expected status client")
	}
	if bundle.Modules.Notifications.NotificationClient == nil {
		t.Fatal("expected notification client")
	}
	if bundle.Modules.Discovery.DiscoveryClient == nil {
		t.Fatal("expected discovery client")
	}
	if bundle.Principal.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Principal.AssetBaseURL = %q, want %q", bundle.Principal.AssetBaseURL, "https://cdn.example.com/assets")
	}
	if bundle.Modules.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Modules.AssetBaseURL = %q, want %q", bundle.Modules.AssetBaseURL, "https://cdn.example.com/assets")
	}
}

func TestBootstrapDependenciesProvideHealthyCampaignsGateway(t *testing.T) {
	cfg := testDependencyConfig()
	requirements := mustDependencyRequirements(t, cfg)
	reporter := platformstatus.NewReporter("web", nil)

	bundle, conns, err := bootstrapDependencies(context.Background(), requirements, "", reporter, nil, testManagedConnFactory(t))
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns, nil)

	gateway := campaigngateway.NewCatalogReadGateway(campaigngateway.CatalogReadDeps{
		Campaign: bundle.Modules.Campaigns.CampaignClient,
	}, bundle.Modules.AssetBaseURL)
	if gateway == nil {
		t.Fatal("expected bootstrapped campaigns gateway to be healthy")
	}
}

func TestBootstrapDependenciesErrorClosesConns(t *testing.T) {
	callCount := 0
	newConn := func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		callCount++
		if cfg.Name == web.DependencyNameGame {
			return nil, errors.New("game unavailable")
		}
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		cfg.StatusReporter = nil
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}

	cfg := testDependencyConfig()
	requirements := mustDependencyRequirements(t, cfg)
	reporter := platformstatus.NewReporter("web", nil)

	_, _, err := bootstrapDependencies(context.Background(), requirements, "", reporter, nil, newConn)
	if err == nil {
		t.Fatal("expected error for game dependency failure")
	}
	if !strings.Contains(err.Error(), "game") {
		t.Fatalf("error = %q, want game dependency detail", err.Error())
	}
	// Game is the 3rd dependency — auth and social succeed, game fails.
	if callCount != 3 {
		t.Fatalf("newManagedConn calls = %d, want 3", callCount)
	}
}

func TestBootstrapDependenciesSkipsEmptyAddress(t *testing.T) {
	cfg := testDependencyConfig()
	cfg.AIAddr = ""
	cfg.DiscoveryAddr = "  "
	requirements := mustDependencyRequirements(t, cfg)
	reporter := platformstatus.NewReporter("web", nil)

	_, conns, err := bootstrapDependencies(context.Background(), requirements, "", reporter, nil, testManagedConnFactory(t))
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns, nil)

	// 9 requirements - 2 empty addresses = 7 connections.
	if len(conns) != 7 {
		t.Fatalf("managed conns = %d, want 7", len(conns))
	}
}

func TestBootstrapRuntimeDependenciesWiresStatusClientIntoDashboard(t *testing.T) {
	t.Parallel()

	cfg := testDependencyConfig()
	reporter := platformstatus.NewReporter("web", nil)

	runtimeDeps, err := bootstrapRuntimeDependencies(context.Background(), cfg, reporter, &bootstrapOptions{NewConn: testManagedConnFactory(t)})
	if err != nil {
		t.Fatalf("bootstrapRuntimeDependencies() error = %v", err)
	}
	defer runtimeDeps.close()

	if runtimeDeps.bundle.Modules.Dashboard.StatusClient == nil {
		t.Fatal("expected dashboard status client")
	}
	if len(runtimeDeps.depsConns) != 9 {
		t.Fatalf("managed conns = %d, want %d", len(runtimeDeps.depsConns), 9)
	}
}

func TestBindStatusReporterSetsClientAfterConnReady(t *testing.T) {
	t.Parallel()

	server := &recordingStatusServer{reports: make(chan *statusv1.ReportStatusRequest, 1)}
	grpcServer := grpc.NewServer()
	statusv1.RegisterStatusServiceServer(grpcServer, server)
	healthServer := grpcHealth.NewServer()
	healthServer.SetServingStatus("", grpcHealthV1.HealthCheckResponse_SERVING)
	grpcHealthV1.RegisterHealthServer(grpcServer, healthServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()

	reporter := platformstatus.NewReporter(
		"web",
		nil,
		platformstatus.WithPushInterval(time.Hour),
		platformstatus.WithLogFunc(func(string, ...any) {}),
	)
	reporter.Register("web.status.integration", platformstatus.Operational)
	reporter.SetDegraded("web.status.integration", "waiting")

	reporterCtx, cancelReporter := context.WithCancel(context.Background())
	stopReporter := reporter.Start(reporterCtx)
	defer func() {
		cancelReporter()
		stopReporter()
	}()

	connCtx, cancelConn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelConn()
	mc, err := platformgrpc.NewManagedConn(connCtx, platformgrpc.ManagedConnConfig{
		Name: "status",
		Addr: listener.Addr().String(),
		Mode: platformgrpc.ModeOptional,
		DialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		Logf: func(string, ...any) {},
	})
	if err != nil {
		t.Fatalf("NewManagedConn() error = %v", err)
	}
	defer func() { _ = mc.Close() }()

	bindStatusReporter(reporter)(connCtx, mc)

	select {
	case req := <-server.reports:
		if req.GetReport().GetService() != "web" {
			t.Fatalf("reported service = %q, want %q", req.GetReport().GetService(), "web")
		}
		if len(req.GetReport().GetCapabilities()) != 1 {
			t.Fatalf("capability count = %d, want 1", len(req.GetReport().GetCapabilities()))
		}
		if req.GetReport().GetCapabilities()[0].GetName() != "web.status.integration" {
			t.Fatalf("capability name = %q, want %q", req.GetReport().GetCapabilities()[0].GetName(), "web.status.integration")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for reporter to flush after status connection became ready")
	}
}

type recordingStatusServer struct {
	statusv1.UnimplementedStatusServiceServer
	reports chan *statusv1.ReportStatusRequest
}

func (s *recordingStatusServer) ReportStatus(_ context.Context, req *statusv1.ReportStatusRequest) (*statusv1.ReportStatusResponse, error) {
	s.reports <- req
	return &statusv1.ReportStatusResponse{}, nil
}

func TestConfigServerConfigMapsRuntimeDependencies(t *testing.T) {
	t.Parallel()

	cfg := Config{
		HTTPAddr:            "127.0.0.1:8080",
		PlayHTTPAddr:        "127.0.0.1:8094",
		TrustForwardedProto: true,
	}
	deps := web.DependencyBundle{}

	serverCfg := cfg.serverConfig(deps, playlaunchgrant.Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      time.Minute,
	})
	if serverCfg.HTTPAddr != cfg.HTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", serverCfg.HTTPAddr, cfg.HTTPAddr)
	}
	if serverCfg.PlayHTTPAddr != cfg.PlayHTTPAddr {
		t.Fatalf("PlayHTTPAddr = %q, want %q", serverCfg.PlayHTTPAddr, cfg.PlayHTTPAddr)
	}
	if !serverCfg.RequestSchemePolicy.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = false, want true")
	}
	if serverCfg.Dependencies == nil {
		t.Fatal("expected dependencies pointer")
	}
	if serverCfg.Logger == nil {
		t.Fatal("expected logger")
	}
}
