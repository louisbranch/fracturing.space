// Package game exposes the stable gRPC surface over core domain aggregates.
//
// Each service maps to a domain aggregate and defines the intent boundary for one
// area of campaign state:
//   - CampaignService -> campaign lifecycle, status transitions, and fork metadata
//   - ParticipantService -> participant roster, roles, and controller access
//   - InviteService -> invite issuance, claim, and revocation workflows
//   - CharacterService -> character ownership and profile state
//   - SessionService -> active session lifecycle, gates, and spotlight
//   - SnapshotService -> replay checkpoints and materialized read state
//   - ForkService -> campaign branching for scenario exploration
//   - EventService -> raw event stream visibility for audit/debug
//   - StatisticsService -> aggregate-level telemetry and counters
package game
