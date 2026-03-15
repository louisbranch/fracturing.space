// Package dashboardsync owns the shared cross-module dashboard refresh policy
// used by web mutations.
//
// Start here when changing how invite/settings/campaign mutations invalidate
// dashboard freshness. This package should stay one shared policy seam built by
// the registry, not a place where feature-specific mutation logic accumulates.
package dashboardsync
