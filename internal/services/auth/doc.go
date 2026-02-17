// Package auth defines the identity boundary used across the platform.
//
// It is the single place that owns user lifecycle, authentication factors,
// and grant issuance so other services can depend on stable user IDs and
// authorization checks instead of re-implementing identity rules.
//
// Subpackages:
//   - app: auth server wiring and lifecycle
//   - api/grpc/auth: gRPC AuthService handlers
//   - oauth: OAuth endpoints, providers, and token flows
//   - storage: persistence interfaces and SQLite implementations
//   - user: user domain model and helpers
package auth
