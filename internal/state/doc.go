// Package state provides game state management across three tiers.
//
// The state package organizes campaign data by change frequency:
//
// # Campaign (Config Layer)
//
// Settings that rarely change after setup: campaign name, system, GM mode,
// status, participant list. These define the "shape" of the campaign.
//
// Subpackages:
//   - state/campaign: Campaign configuration and lifecycle
//   - state/participant: Player and GM participant management
//   - state/character: Character definitions and profiles
//
// # Snapshot (Continuity Layer)
//
// State that changes between sessions: character state (HP, Hope, Stress),
// GM Fear resource, story progress, quest completion. This is the "save game"
// data that persists across sessions.
//
// Subpackage:
//   - state/snapshot: Cross-session continuity state
//
// # Session (Gameplay Layer)
//
// State that changes every action: active session, events, pending rolls,
// outcomes. This is the real-time gameplay data.
//
// Subpackage:
//   - state/session: Session lifecycle and events
//
// # Design Philosophy
//
// This three-tier model helps developers understand where to put new features:
//   - Adding campaign settings? → state/campaign
//   - Adding persistent gameplay values? → state/snapshot
//   - Adding session mechanics? → state/session
package state
