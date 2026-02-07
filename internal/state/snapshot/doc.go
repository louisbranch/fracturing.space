// Package snapshot provides cross-session continuity state management.
//
// This is the "continuity layer" of state management - data that persists
// across sessions but changes during gameplay: character state (HP, Hope,
// Stress), GM Fear resource, story progress, quest completion.
//
// # Snapshot Model
//
// The Snapshot aggregate groups all continuity state for a campaign:
//   - Character states (HP, Hope, Stress for each character)
//   - GM Fear (campaign-level resource)
//   - Future: story progress, quest completion, world state
//
// # Design Philosophy
//
// Snapshot state represents the "save game" - if a session is interrupted,
// snapshot state captures what needs to persist. Campaign state (name, system,
// participants) rarely changes; session state (events, rolls) is ephemeral.
// Snapshot sits in between.
//
// # GM Fear
//
// GM Fear was previously stored in the Campaign entity but is now part of
// Snapshot because it changes during gameplay sessions, not during campaign
// setup.
package snapshot
