package modules

import (
	"strings"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
)

// DeriveServiceHealth builds health entries from modules that implement
// HealthReporter. Each module is the single source of truth for its own
// availability — new dependencies automatically affect health without
// manual registry updates.
func DeriveServiceHealth(modules []Module) []dashboard.ServiceHealthEntry {
	var entries []dashboard.ServiceHealthEntry
	for _, m := range modules {
		hr, ok := m.(module.HealthReporter)
		if !ok {
			continue
		}
		entries = append(entries, dashboard.ServiceHealthEntry{
			Label:     capitalizeLabel(m.ID()),
			Available: hr.Healthy(),
		})
	}
	return entries
}

func capitalizeLabel(id string) string {
	if id == "" {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
}
