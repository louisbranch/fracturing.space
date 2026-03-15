package modules

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"google.golang.org/grpc"
)

func TestDefaultModulesIncludeOnlyStableAreas(t *testing.T) {
	t.Parallel()

	reg := NewRegistryBuilder()
	built := reg.Build(RegistryInput{
		Dependencies:     Dependencies{},
		Principal:        principal.Principal{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	public := built.Public
	protected := built.Protected
	if len(public) != 6 {
		t.Fatalf("public module count = %d, want %d", len(public), 6)
	}
	if len(protected) != 2 {
		t.Fatalf("protected module count = %d, want %d", len(protected), 2)
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
	if got := public[5].ID(); got != "invite" {
		t.Fatalf("default public module[5] id = %q, want %q", got, "invite")
	}
	if got := protected[0].ID(); got != "dashboard" {
		t.Fatalf("default protected module[0] id = %q, want %q", got, "dashboard")
	}
	if got := protected[1].ID(); got != "settings" {
		t.Fatalf("default protected module[1] id = %q, want %q", got, "settings")
	}
}

func TestDefaultProtectedModulesDelegatesToBuilder(t *testing.T) {
	t.Parallel()

	deps := Dependencies{}
	principal := principal.Principal{}
	opts := ProtectedModuleOptions{}

	modules := defaultProtectedModules(deps, principal, opts)
	builtModules := buildProtectedModules(deps, principal, opts)
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

	reg := NewRegistryBuilder()
	built := reg.Build(RegistryInput{
		Dependencies:     Dependencies{},
		Principal:        principal.Principal{},
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

func TestRegistryBuildComposesExpectedModules(t *testing.T) {
	t.Parallel()

	reg := NewRegistryBuilder()
	built := reg.Build(RegistryInput{
		Dependencies:     Dependencies{},
		Principal:        principal.Principal{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	if len(built.Public) != 6 {
		t.Fatalf("public module count = %d, want 6", len(built.Public))
	}
	if len(built.Protected) != 2 {
		t.Fatalf("protected module count = %d, want 2", len(built.Protected))
	}
}

func TestRegistryBuildIncludesNotificationsWhenClientConfigured(t *testing.T) {
	t.Parallel()

	reg := NewRegistryBuilder()
	built := reg.Build(RegistryInput{
		Dependencies: Dependencies{
			Notifications: NotificationDependencies{
				NotificationClient: stubNotificationClient{},
			},
		},
		Principal:        principal.Principal{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	if len(built.Protected) != 3 {
		t.Fatalf("protected module count = %d, want 3", len(built.Protected))
	}
	if got := built.Protected[2].ID(); got != "notifications" {
		t.Fatalf("protected module[2] id = %q, want %q", got, "notifications")
	}
}

func TestRegistryBuildIncludesCampaignsWhenDependencySetIsComplete(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	deps := NewDependencies("https://cdn.example.com/assets")
	BindAuthDependency(&deps, conn)
	BindSocialDependency(&deps, conn)
	BindGameDependency(&deps, conn)
	BindAIDependency(&deps, conn)
	BindDiscoveryDependency(&deps, conn)

	reg := NewRegistryBuilder()
	built := reg.Build(RegistryInput{
		Dependencies:     deps,
		Principal:        principal.Principal{},
		PublicOptions:    PublicModuleOptions{},
		ProtectedOptions: ProtectedModuleOptions{},
	})
	if len(built.Protected) != 3 {
		t.Fatalf("protected module count = %d, want 3", len(built.Protected))
	}
	if got := built.Protected[2].ID(); got != "campaigns" {
		t.Fatalf("protected module[2] id = %q, want %q", got, "campaigns")
	}
}

func TestNewSharedServicesProvidesNoopDashboardSyncWhenDepsMissing(t *testing.T) {
	t.Parallel()

	shared := newSharedServices(Dependencies{})
	if shared.dashboardSync == nil {
		t.Fatal("dashboardSync = nil, want no-op service")
	}

	shared.dashboardSync.ProfileSaved(context.Background(), "user-1")
	shared.dashboardSync.CampaignCreated(context.Background(), "user-1", "camp-1")
	shared.dashboardSync.SessionStarted(context.Background(), "user-1", "camp-1")
	shared.dashboardSync.SessionEnded(context.Background(), "user-1", "camp-1")
	shared.dashboardSync.InviteChanged(context.Background(), []string{"user-1"}, "camp-1")
}

func TestNewSharedServicesReturnsSyncerWhenDashboardDependenciesConfigured(t *testing.T) {
	t.Parallel()

	shared := newSharedServices(Dependencies{
		DashboardSync: DashboardSyncDependencies{
			UserHubControlClient: stubUserHubControlClient{},
			GameEventClient:      stubGameEventClient{},
		},
	})
	if _, ok := shared.dashboardSync.(*dashboardsync.Syncer); !ok {
		t.Fatalf("dashboardSync = %T, want *dashboardsync.Syncer", shared.dashboardSync)
	}
}

func TestCapitalizeLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "game", want: "Game"},
		{input: "Game", want: "Game"},
		{input: "userhub", want: "Userhub"},
	}
	for _, tc := range tests {
		if got := capitalizeLabel(tc.input); got != tc.want {
			t.Fatalf("capitalizeLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// stubStatusClient implements statusv1.StatusServiceClient for unit tests.
type stubStatusClient struct {
	statusv1.StatusServiceClient
	resp *statusv1.GetSystemStatusResponse
	err  error
}

type stubNotificationClient struct{}

type stubUserHubControlClient struct{}

type stubGameEventClient struct{}

func (stubNotificationClient) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}

func (stubNotificationClient) ListNotifications(context.Context, *notificationsv1.ListNotificationsRequest, ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	return &notificationsv1.ListNotificationsResponse{}, nil
}

func (stubNotificationClient) GetNotification(_ context.Context, req *notificationsv1.GetNotificationRequest, _ ...grpc.CallOption) (*notificationsv1.GetNotificationResponse, error) {
	return &notificationsv1.GetNotificationResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

func (stubNotificationClient) MarkNotificationRead(_ context.Context, req *notificationsv1.MarkNotificationReadRequest, _ ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	return &notificationsv1.MarkNotificationReadResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

func (stubUserHubControlClient) InvalidateDashboards(context.Context, *userhubv1.InvalidateDashboardsRequest, ...grpc.CallOption) (*userhubv1.InvalidateDashboardsResponse, error) {
	return &userhubv1.InvalidateDashboardsResponse{}, nil
}

func (stubGameEventClient) ListEvents(context.Context, *gamev1.ListEventsRequest, ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
	return &gamev1.ListEventsResponse{}, nil
}

func (stubGameEventClient) SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	return nil, nil
}

func (s *stubStatusClient) GetSystemStatus(_ context.Context, _ *statusv1.GetSystemStatusRequest, _ ...grpc.CallOption) (*statusv1.GetSystemStatusResponse, error) {
	return s.resp, s.err
}

func TestStatusHealthProviderNilClient(t *testing.T) {
	t.Parallel()

	provider := dashboard.StatusHealthProvider(nil, nil)
	if provider != nil {
		t.Fatal("expected nil provider for nil client")
	}
}

func TestStatusHealthProviderReturnsEntries(t *testing.T) {
	t.Parallel()

	client := &stubStatusClient{
		resp: &statusv1.GetSystemStatusResponse{
			Services: []*statusv1.ServiceStatus{
				{
					Service:         "userhub",
					AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED,
				},
				{
					Service:         "game",
					AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
				},
				nil, // nil entry should be skipped
			},
		},
	}

	provider := dashboard.StatusHealthProvider(client, nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	entries := provider(context.Background())
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(entries))
	}
	// Entries should be sorted alphabetically.
	if entries[0].Label != "Game" {
		t.Fatalf("entries[0].Label = %q, want %q", entries[0].Label, "Game")
	}
	if !entries[0].Available {
		t.Fatal("entries[0].Available = false, want true")
	}
	if entries[1].Label != "Userhub" {
		t.Fatalf("entries[1].Label = %q, want %q", entries[1].Label, "Userhub")
	}
	if entries[1].Available {
		t.Fatal("entries[1].Available = true, want false")
	}
}

func TestStatusHealthProviderErrorReturnsNil(t *testing.T) {
	t.Parallel()

	client := &stubStatusClient{
		err: context.DeadlineExceeded,
	}
	provider := dashboard.StatusHealthProvider(client, nil)
	entries := provider(context.Background())
	if entries != nil {
		t.Fatalf("expected nil entries on error, got %d", len(entries))
	}
}

func TestStatusHealthProviderEmptyServicesReturnsNil(t *testing.T) {
	t.Parallel()

	client := &stubStatusClient{
		resp: &statusv1.GetSystemStatusResponse{},
	}
	provider := dashboard.StatusHealthProvider(client, nil)
	entries := provider(context.Background())
	if entries != nil {
		t.Fatalf("expected nil entries for empty services, got %d", len(entries))
	}
}

// Verify ServiceHealthEntry is properly populated by checking the type contract.
var _ = []dashboardapp.ServiceHealthEntry{}
