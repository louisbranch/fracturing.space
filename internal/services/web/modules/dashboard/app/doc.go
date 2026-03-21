// Package app contains dashboard domain contracts and orchestration logic.
//
// Start here when changing how dashboard cards, freshness state, or web-owned
// aggregation rules are assembled for transport. Gateway adapters should stay
// in `dashboard/gateway`; this package owns the small service seam that keeps
// the root dashboard module transport-thin.
package app
