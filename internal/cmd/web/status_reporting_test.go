package web

import (
	"testing"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
)

func TestRegisterDependencyCapabilitiesMapsStates(t *testing.T) {
	t.Parallel()

	requirements := dependencyRequirements(testDependencyConfig())
	statuses := map[string]dependencyStatus{
		dependencyNameAuth: {
			Name:    dependencyNameAuth,
			Address: "auth:8083",
			State:   dependencyDialStateConnected,
		},
		dependencyNameSocial: {
			Name:    dependencyNameSocial,
			Address: "social:8090",
			State:   dependencyDialStateDialFailed,
		},
		dependencyNameGame: {
			Name:    dependencyNameGame,
			Address: "game:8082",
			State:   dependencyDialStateUnavailable,
		},
	}

	reporter := platformstatus.NewReporter("web", nil)
	registerDependencyCapabilities(reporter, requirements, statuses)

	snapshot := reporter.Snapshot()
	if len(snapshot) != len(requirements) {
		t.Fatalf("registered capabilities = %d, want %d", len(snapshot), len(requirements))
	}

	capabilities := make(map[string]platformstatus.CapabilityStatus, len(snapshot))
	for _, capability := range snapshot {
		capabilities[capability.Name] = capability.Status
	}

	for _, dep := range requirements {
		got, ok := capabilities[dep.capability]
		if !ok {
			t.Fatalf("missing registered capability %q", dep.capability)
		}
		want := platformstatus.Unavailable
		if dep.name == dependencyNameAuth {
			want = platformstatus.Operational
		}
		if got != want {
			t.Fatalf("capability %q status = %v, want %v", dep.capability, got, want)
		}
	}
}

func TestRegisterDependencyCapabilitiesNilReporter(t *testing.T) {
	t.Parallel()

	registerDependencyCapabilities(nil, dependencyRequirements(testDependencyConfig()), map[string]dependencyStatus{})
}
