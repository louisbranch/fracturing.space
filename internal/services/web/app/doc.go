// Package app owns root HTTP composition policy.
//
// Start here when changing how mounted modules are grouped, wrapped by auth,
// normalized for slash redirects, or protected by same-origin checks. This
// package should stay transport-only and policy-focused:
//   - it assembles mounted handlers,
//   - it enforces root mux rules,
//   - it does not own request-state resolution or feature-local routes.
//
// Module startup wiring belongs above this package in composition/, while
// feature-specific behavior belongs below it in modules/<area>/.
package app
