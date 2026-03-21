// Package dashboard owns authenticated dashboard transport routes.
//
// Start here when changing dashboard route ownership, page assembly, or the
// degraded-mode contract for userhub-backed data. The root package owns
// transport, `dashboard/app` owns view orchestration, and `dashboard/gateway`
// maps userhub transport details into the app seam.
//
// Runtime status-health wiring is intentionally area-owned in `composition.go`
// so the central module registry only decides ordering and shared handler base
// inputs.
package dashboard
