package app

import "context"

// ServiceHealthEntry represents the availability status of a backend service group.
type ServiceHealthEntry struct {
	Label     string
	Available bool
}

// HealthProvider returns current service health entries.
// Called on each dashboard load to get live status.
type HealthProvider func(ctx context.Context) []ServiceHealthEntry
