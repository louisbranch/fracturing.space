package modules

import (
	"context"
	"testing"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"google.golang.org/grpc"
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
	builtModules := buildProtectedModules(deps, resolvers, opts)
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

func (s *stubStatusClient) GetSystemStatus(_ context.Context, _ *statusv1.GetSystemStatusRequest, _ ...grpc.CallOption) (*statusv1.GetSystemStatusResponse, error) {
	return s.resp, s.err
}

func TestStatusHealthProviderNilClient(t *testing.T) {
	t.Parallel()

	provider := statusHealthProvider(nil)
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

	provider := statusHealthProvider(client)
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
	provider := statusHealthProvider(client)
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
	provider := statusHealthProvider(client)
	entries := provider(context.Background())
	if entries != nil {
		t.Fatalf("expected nil entries for empty services, got %d", len(entries))
	}
}

// Verify ServiceHealthEntry is properly populated by checking the type contract.
var _ = []dashboard.ServiceHealthEntry{}
