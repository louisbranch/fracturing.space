// Package web parses command inputs and boots the browser-facing web service.
//
// Start here when changing startup flags, environment config, dependency
// address policy, or runtime bootstrap sequencing for the `cmd/web` entrypoint.
// Service-owned startup validation lives in `internal/services/web`; this
// package should stay focused on command config, dependency address resolution,
// managed connection lifecycle, and process boot orchestration.
//
// Important entrypoints:
//   - `web.go` for config parsing and top-level Run flow,
//   - `dependency_graph.go` for startup dependency address policy,
//   - `runtime_dependencies.go` for assembling the runtime dependency bundle.
package web
