// Package auth defines the authentication boundary for Fracturing.Space.
//
// It is an umbrella package for the auth server, OAuth flows, user model,
// and storage implementations used by the gRPC auth service.
//
// Subpackages:
//   - app: auth server wiring and lifecycle
//   - api/grpc/auth: gRPC AuthService handlers
//   - oauth: OAuth endpoints, providers, and token flows
//   - storage: persistence interfaces and SQLite implementations
//   - user: user domain model and helpers
package auth
