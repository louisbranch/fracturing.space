// Package systems contains game system implementations.
//
// Each game system (Daggerheart, D&D 5e, VtM, etc.) is implemented as a
// subpackage that provides system-specific mechanics built on top of the
// generic primitives in internal/core.
//
// # Architecture
//
// The systems layer follows a plugin-like architecture:
//
//   - Each system implements the GameSystem interface
//   - Systems are registered in the registry at startup
//   - Campaigns are bound to a single system at creation time
//   - The API layer uses the registry to dispatch to the correct system
//
// # Adding a New System
//
// To add a new game system:
//
//  1. Create a new subpackage (e.g., internal/systems/dnd5e/)
//  2. Implement the domain types and mechanics
//  3. Register the system in internal/systems/registry.go
//  4. Add proto definitions in api/proto/systems/{name}/v1/
//  5. Create gRPC service handlers in internal/api/grpc/systems/{name}/
//
// See the daggerheart package for a complete reference implementation.
package systems
