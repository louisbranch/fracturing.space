package admin

import (
	"fmt"
	"sort"

	"github.com/louisbranch/fracturing.space/internal/platform/status"
	adminservice "github.com/louisbranch/fracturing.space/internal/services/admin"
)

const (
	adminDependencyGame = "game"
	adminDependencyAuth = "auth"
)

var adminDependencyCapabilities = map[string]string{
	adminDependencyGame: "admin.game.integration",
	adminDependencyAuth: "admin.auth.integration",
}

// formatDependencyStatusLines renders deterministic startup diagnostics.
func formatDependencyStatusLines(statuses []adminservice.DependencyStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	ordered := append([]adminservice.DependencyStatus(nil), statuses...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	lines := make([]string, 0, len(ordered))
	for _, dep := range ordered {
		state := "unavailable"
		if dep.Connected {
			state = "connected"
		}
		lines = append(lines, fmt.Sprintf("dependency=%s state=%s address=%s", dep.Name, state, dep.Address))
	}
	return lines
}

// dependencyStatusWarnings renders deterministic warning strings for unavailable dependencies.
func dependencyStatusWarnings(statuses []adminservice.DependencyStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	ordered := append([]adminservice.DependencyStatus(nil), statuses...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	warnings := make([]string, 0, len(ordered))
	for _, dep := range ordered {
		if dep.Connected {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable", dep.Name, dep.Address))
	}
	return warnings
}

// registerDependencyCapabilities keeps capability registration deterministic and fail-closed.
func registerDependencyCapabilities(reporter *status.Reporter, statuses []adminservice.DependencyStatus) {
	if reporter == nil {
		return
	}
	reporter.Register("admin.dashboard", status.Operational)

	connectivityByName := make(map[string]bool, len(statuses))
	for _, dep := range statuses {
		connectivityByName[dep.Name] = dep.Connected
	}

	capabilityNames := make([]string, 0, len(adminDependencyCapabilities))
	for _, capability := range adminDependencyCapabilities {
		capabilityNames = append(capabilityNames, capability)
	}
	sort.Strings(capabilityNames)

	capabilityToDependency := make(map[string]string, len(adminDependencyCapabilities))
	for dependency, capability := range adminDependencyCapabilities {
		capabilityToDependency[capability] = dependency
	}

	for _, capability := range capabilityNames {
		dependencyName := capabilityToDependency[capability]
		if connectivityByName[dependencyName] {
			reporter.Register(capability, status.Operational)
			continue
		}
		reporter.Register(capability, status.Unavailable)
	}
}
