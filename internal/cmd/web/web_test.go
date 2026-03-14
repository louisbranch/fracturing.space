package web

import (
	"context"
	"errors"
	"flag"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpcHealth "google.golang.org/grpc/health"
	grpcHealthV1 "google.golang.org/grpc/health/grpc_health_v1"
)

func stubManagedConn(t *testing.T) {
	t.Helper()
	previous := newManagedConn
	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		cfg.Mode = platformgrpc.ModeOptional
		cfg.DialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		cfg.StatusReporter = nil
		cfg.Logf = func(string, ...any) {}
		return platformgrpc.NewManagedConn(ctx, cfg)
	}
	t.Cleanup(func() {
		newManagedConn = previous
	})
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
	if cfg.ChatHTTPAddr != "localhost:8086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "localhost:8086")
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

func TestParseConfigOverrideChatHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-chat-http-addr", "127.0.0.1:18086"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.ChatHTTPAddr != "127.0.0.1:18086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "127.0.0.1:18086")
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

func testDependencyConfig() Config {
	return Config{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		GameAddr:          "game:8082",
		AIAddr:            "ai:8087",
		DiscoveryAddr:     "discovery:8091",
		UserHubAddr:       "userhub:8092",
		NotificationsAddr: "notifications:8088",
	}
}

func TestDependencyRequirementsRequiredPolicy(t *testing.T) {
	t.Parallel()

	requirements := dependencyRequirements(testDependencyConfig(), nil)
	requiredNames := make([]string, 0, len(requirements))
	for _, dep := range requirements {
		if dep.policy == startupDependencyRequired {
			requiredNames = append(requiredNames, dep.name)
		}
	}
	slices.Sort(requiredNames)
	want := []string{dependencyNameAuth, dependencyNameGame, dependencyNameSocial}
	if !slices.Equal(requiredNames, want) {
		t.Fatalf("required dependencies = %v, want %v", requiredNames, want)
	}
}

func TestDependencyRequirementsOwnedSurfacesAreExplicit(t *testing.T) {
	t.Parallel()

	requirements := dependencyRequirements(testDependencyConfig(), nil)
	got := map[string][]string{}
	for _, dep := range requirements {
		if len(dep.surfaces) == 0 {
			t.Fatalf("dependency %q has no owned surfaces", dep.name)
		}
		got[dep.name] = dep.surfaces
	}

	tests := map[string][]string{
		dependencyNameAuth:          {"principal", "publicauth", "profile", "settings"},
		dependencyNameSocial:        {"principal", "profile", "settings", "campaigns"},
		dependencyNameGame:          {"campaigns", "dashboard-sync"},
		dependencyNameAI:            {"settings.ai", "campaigns.ai"},
		dependencyNameDiscovery:     {"discovery"},
		dependencyNameUserHub:       {"dashboard", "dashboard-sync"},
		dependencyNameNotifications: {"principal", "notifications"},
		dependencyNameStatus:        {"dashboard.health"},
	}
	for name, want := range tests {
		if !slices.Equal(got[name], want) {
			t.Fatalf("dependency %q surfaces = %v, want %v", name, got[name], want)
		}
	}
}

func TestDependencyRequirementsCapabilitiesAreUnique(t *testing.T) {
	t.Parallel()

	requirements := dependencyRequirements(testDependencyConfig(), nil)
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

	if got := startupDependencyRequired.managedConnMode(); got != platformgrpc.ModeRequired {
		t.Fatalf("required managedConnMode = %v, want %v", got, platformgrpc.ModeRequired)
	}
	if got := startupDependencyOptional.managedConnMode(); got != platformgrpc.ModeOptional {
		t.Fatalf("optional managedConnMode = %v, want %v", got, platformgrpc.ModeOptional)
	}
}

func TestBootstrapDependenciesWiresAllClients(t *testing.T) {
	stubManagedConn(t)

	cfg := testDependencyConfig()
	cfg.AssetBaseURL = "https://cdn.example.com/assets"
	requirements := dependencyRequirements(cfg, nil)
	reporter := platformstatus.NewReporter("web", nil)

	bundle, conns, err := bootstrapDependencies(context.Background(), requirements, cfg.AssetBaseURL, reporter)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns)

	if len(conns) != 7 {
		t.Fatalf("managed conns = %d, want 7", len(conns))
	}
	if bundle.Principal.SessionClient == nil {
		t.Fatal("expected principal session client")
	}
	if bundle.Modules.Campaigns.CampaignClient == nil {
		t.Fatal("expected campaign client")
	}
	if bundle.Modules.Campaigns.CommunicationClient == nil {
		t.Fatal("expected campaign communication client")
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
	stubManagedConn(t)

	cfg := testDependencyConfig()
	requirements := dependencyRequirements(cfg, nil)
	reporter := platformstatus.NewReporter("web", nil)

	bundle, conns, err := bootstrapDependencies(context.Background(), requirements, "", reporter)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns)

	gateway := campaigngateway.NewGRPCGateway(campaigngateway.GRPCGatewayDeps{
		Read: campaigngateway.GRPCGatewayReadDeps{
			Campaign:           bundle.Modules.Campaigns.CampaignClient,
			Communication:      bundle.Modules.Campaigns.CommunicationClient,
			Agent:              bundle.Modules.Campaigns.AgentClient,
			Participant:        bundle.Modules.Campaigns.ParticipantClient,
			Character:          bundle.Modules.Campaigns.CharacterClient,
			DaggerheartContent: bundle.Modules.Campaigns.DaggerheartContentClient,
			DaggerheartAsset:   bundle.Modules.Campaigns.DaggerheartAssetClient,
			Session:            bundle.Modules.Campaigns.SessionClient,
			Invite:             bundle.Modules.Campaigns.InviteClient,
			Social:             bundle.Modules.Campaigns.SocialClient,
		},
		Mutation: campaigngateway.GRPCGatewayMutationDeps{
			Campaign:    bundle.Modules.Campaigns.CampaignClient,
			Participant: bundle.Modules.Campaigns.ParticipantClient,
			Character:   bundle.Modules.Campaigns.CharacterClient,
			Session:     bundle.Modules.Campaigns.SessionClient,
			Invite:      bundle.Modules.Campaigns.InviteClient,
			Auth:        bundle.Modules.Campaigns.AuthClient,
		},
		Authorization: campaigngateway.GRPCGatewayAuthorizationDeps{
			Client: bundle.Modules.Campaigns.AuthorizationClient,
		},
	})
	if !campaignapp.IsGatewayHealthy(gateway) {
		t.Fatal("expected bootstrapped campaigns gateway to be healthy")
	}
}

func TestBootstrapDependenciesErrorClosesConns(t *testing.T) {
	previous := newManagedConn
	callCount := 0
	newManagedConn = func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		callCount++
		if cfg.Name == dependencyNameGame {
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
	t.Cleanup(func() {
		newManagedConn = previous
	})

	cfg := testDependencyConfig()
	requirements := dependencyRequirements(cfg, nil)
	reporter := platformstatus.NewReporter("web", nil)

	_, _, err := bootstrapDependencies(context.Background(), requirements, "", reporter)
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
	stubManagedConn(t)

	cfg := testDependencyConfig()
	cfg.AIAddr = ""
	cfg.DiscoveryAddr = "  "
	requirements := dependencyRequirements(cfg, nil)
	reporter := platformstatus.NewReporter("web", nil)

	_, conns, err := bootstrapDependencies(context.Background(), requirements, "", reporter)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	defer closeManagedConns(conns)

	// 8 requirements - 3 empty addresses = 5 connections.
	if len(conns) != 5 {
		t.Fatalf("managed conns = %d, want 5", len(conns))
	}
}

func TestBootstrapRuntimeDependenciesWiresStatusClientIntoDashboard(t *testing.T) {
	t.Parallel()

	stubManagedConn(t)

	cfg := testDependencyConfig()
	cfg.StatusAddr = "status:8093"
	reporter := platformstatus.NewReporter("web", nil)

	runtimeDeps, err := bootstrapRuntimeDependencies(context.Background(), cfg, reporter)
	if err != nil {
		t.Fatalf("bootstrapRuntimeDependencies() error = %v", err)
	}
	defer runtimeDeps.close()

	if runtimeDeps.bundle.Modules.Dashboard.StatusClient == nil {
		t.Fatal("expected dashboard status client")
	}
	if len(runtimeDeps.depsConns) != 8 {
		t.Fatalf("managed conns = %d, want %d", len(runtimeDeps.depsConns), 8)
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
		ChatHTTPAddr:        "127.0.0.1:8086",
		TrustForwardedProto: true,
	}
	deps := web.DependencyBundle{}

	serverCfg := cfg.serverConfig(deps)
	if serverCfg.HTTPAddr != cfg.HTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", serverCfg.HTTPAddr, cfg.HTTPAddr)
	}
	if serverCfg.ChatHTTPAddr != cfg.ChatHTTPAddr {
		t.Fatalf("ChatHTTPAddr = %q, want %q", serverCfg.ChatHTTPAddr, cfg.ChatHTTPAddr)
	}
	if !serverCfg.RequestSchemePolicy.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = false, want true")
	}
	if serverCfg.Dependencies == nil {
		t.Fatal("expected dependencies pointer")
	}
}
