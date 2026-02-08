// Package campaign provides campaign aggregate gRPC services.
//
// These services handle campaign configuration, session lifecycle, participant
// management, character profiles, and snapshot (continuity) state. They work
// identically regardless of which game system a campaign uses.
//
// # Services
//
//   - CampaignService: Campaign CRUD and lifecycle management
//   - SessionService: Session start/end and event management
//   - ParticipantService: Player and GM participant management
//   - CharacterService: Character and profile management
//   - SnapshotService: Cross-session continuity state (character state, GM fear)
//
// # Usage Pattern
//
// System-specific services (like DaggerheartService) call these campaign services
// to persist gameplay effects. For example, DaggerheartService.ApplyRollOutcome
// calls SnapshotService to update character hope/stress and GM fear.
package campaign
