// Package session serves as an umbrella for game session management functionality,
// including session lifecycle tracking and gRPC services.
//
// The package is organized into two primary subpackages:
//   - domain: Defines the session entity and its lifecycle states (Active, Paused, Ended).
//   - service: Implements the gRPC API layer for starting, listing, and managing sessions.
package session
