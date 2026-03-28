// Package web owns the browser-facing service root.
//
// Start here when changing startup validation, dependency assembly contracts,
// root handler construction, or the boundary between request-scoped principal
// resolution and module registry assembly.
//
// This package should stay small and reader-first:
//   - production startup contracts live here and in internal/cmd/web,
//   - request/mount policy lives below this package in app/ and principal/,
//   - feature-local route and backend wiring belongs under modules/.
//
// Tests that validate the root web runtime also start here, especially
// server_test.go plus the explicit test-harness helpers/defaults.
package web
