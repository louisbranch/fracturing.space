// Package snapshot provides materialized projections derived from the event journal.
//
// Snapshots are captured at a specific event sequence to speed up replay and
// rebuilds. They are not authoritative.
//
// # Snapshot Model
//
// The Snapshot aggregate groups all snapshot data for a campaign:
//   - Character states (HP, Hope, Stress for each character)
//   - GM Fear (campaign-level resource)
//   - Future: story progress, quest completion, world state
//
// # Design Philosophy
//
// Snapshot data represents a materialized projection of the event journal used
// for fast rebuilds. Campaign state (name, system, participants) is handled by
// projections derived from events; session state (events, rolls) remains in the
// journal itself.
//
// # GM Fear
//
// GM Fear was previously stored in the Campaign entity but is now part of
// Snapshot because it changes during gameplay sessions, not during campaign
// setup.
package snapshot
