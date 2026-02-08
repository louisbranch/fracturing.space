// Package campaign provides the campaign aggregate and its supporting areas.
//
// The campaign aggregate is the bucket for configuration, participants,
// characters, events, and gameplay continuity. Events are first-class records
// in the campaign journal; projections and snapshots are derived views.
//
// The package organizes campaign data by change frequency:
//
// # Campaign (Config Layer)
//
// Settings that rarely change after setup: campaign name, system, GM mode,
// status, participant list. These define the "shape" of the campaign.
//
// Subpackages:
//   - campaign/participant: Player and GM participant management
//   - campaign/character: Character definitions and profiles
//
// # Snapshot (Continuity Layer)
//
// State that changes between sessions: character state (HP, Hope, Stress),
// GM Fear resource, story progress, quest completion. This is the "save game"
// data that persists across sessions.
//
// Subpackage:
//   - campaign/snapshot: Cross-session continuity state
//
// # Session (Gameplay Layer)
//
// State that changes every action: active session, events, pending rolls,
// outcomes. This is the real-time gameplay data.
//
// Subpackage:
//   - campaign/session: Session lifecycle and events
//
// # Game System
//
// Each campaign is bound to exactly one game system (Daggerheart, D&D 5e, etc.)
// at creation time. This determines which mechanics are available and how
// the MCP exposes tools.
//
// # GM Fear
//
// Note: GM Fear is now managed in the snapshot package as part of continuity
// state, not campaign configuration.
package campaign
