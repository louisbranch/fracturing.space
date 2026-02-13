// Package campaign provides the campaign aggregate and supporting domain areas.
//
// Campaigns are represented by metadata plus event-derived projections. The
// package houses core campaign lifecycle policies and the domain models used
// by storage and gRPC layers.
//
// Subpackages include:
//   - character: character definitions and profiles
//   - participant: participant membership and roles
//   - session: session lifecycle and outcomes
//   - event: event journal types and normalization
//   - projection: event replay and projection helpers
//   - snapshot: materialized projections (GM fear and system state)
//   - fork: campaign forking metadata and validation
//   - invite: campaign invitation modeling
//   - policy: permission and lifecycle policy helpers
package campaign
