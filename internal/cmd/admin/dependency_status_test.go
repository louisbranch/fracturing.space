package admin

import (
	"reflect"
	"slices"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/status"
	adminservice "github.com/louisbranch/fracturing.space/internal/services/admin"
)

func TestFormatDependencyStatusLinesSorted(t *testing.T) {
	t.Parallel()

	lines := formatDependencyStatusLines([]adminservice.DependencyStatus{
		{Name: "game", Address: "game:8082", Connected: false},
		{Name: "auth", Address: "auth:8083", Connected: true},
	})
	want := []string{
		"dependency=auth state=connected address=auth:8083",
		"dependency=game state=unavailable address=game:8082",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("formatDependencyStatusLines() = %#v, want %#v", lines, want)
	}
}

func TestDependencyStatusWarnings(t *testing.T) {
	t.Parallel()

	warnings := dependencyStatusWarnings([]adminservice.DependencyStatus{
		{Name: "status", Address: "status:8093", Connected: false},
		{Name: "auth", Address: "auth:8083", Connected: true},
		{Name: "game", Address: "game:8082", Connected: false},
	})
	want := []string{
		"game dependency at game:8082 unavailable",
		"status dependency at status:8093 unavailable",
	}
	if !reflect.DeepEqual(warnings, want) {
		t.Fatalf("dependencyStatusWarnings() = %#v, want %#v", warnings, want)
	}
}

func TestRegisterDependencyCapabilities(t *testing.T) {
	t.Parallel()

	reporter := status.NewReporter("admin", nil)
	registerDependencyCapabilities(reporter, []adminservice.DependencyStatus{
		{Name: "game", Address: "game:8082", Connected: true},
		{Name: "auth", Address: "auth:8083", Connected: false},
		{Name: "status", Address: "status:8093", Connected: true},
	})

	snapshot := reporter.Snapshot()
	if len(snapshot) != 3 {
		t.Fatalf("len(snapshot) = %d, want 3", len(snapshot))
	}

	byName := map[string]status.CapabilityStatus{}
	for _, capability := range snapshot {
		byName[capability.Name] = capability.Status
	}
	if byName["admin.dashboard"] != status.Operational {
		t.Fatalf("admin.dashboard status = %v, want %v", byName["admin.dashboard"], status.Operational)
	}
	if byName["admin.game.integration"] != status.Operational {
		t.Fatalf("admin.game.integration status = %v, want %v", byName["admin.game.integration"], status.Operational)
	}
	if byName["admin.auth.integration"] != status.Unavailable {
		t.Fatalf("admin.auth.integration status = %v, want %v", byName["admin.auth.integration"], status.Unavailable)
	}
}

func TestRegisterDependencyCapabilitiesFailClosedWhenMissing(t *testing.T) {
	t.Parallel()

	reporter := status.NewReporter("admin", nil)
	registerDependencyCapabilities(reporter, nil)

	snapshot := reporter.Snapshot()
	names := make([]string, 0, len(snapshot))
	byName := map[string]status.CapabilityStatus{}
	for _, capability := range snapshot {
		names = append(names, capability.Name)
		byName[capability.Name] = capability.Status
	}
	slices.Sort(names)
	wantNames := []string{"admin.auth.integration", "admin.dashboard", "admin.game.integration"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("capability names = %#v, want %#v", names, wantNames)
	}
	if byName["admin.auth.integration"] != status.Unavailable {
		t.Fatalf("admin.auth.integration status = %v, want %v", byName["admin.auth.integration"], status.Unavailable)
	}
	if byName["admin.game.integration"] != status.Unavailable {
		t.Fatalf("admin.game.integration status = %v, want %v", byName["admin.game.integration"], status.Unavailable)
	}
}
