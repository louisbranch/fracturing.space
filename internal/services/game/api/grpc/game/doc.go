// Package game provides campaign aggregate gRPC services.
//
// These services handle campaign configuration, session lifecycle, participant
// management, character profiles, and snapshot projections. They work
// identically regardless of which game system a campaign uses.
//
// # Services
//
//   - CampaignService: Campaign CRUD and lifecycle management
//   - SessionService: Session start/end and event management
//   - ParticipantService: Player and GM participant management
//   - CharacterService: Character and profile management
//   - SnapshotService: Snapshot projections derived from the event journal
//
// # Usage Pattern
//
// System-specific services (like DaggerheartService) call these campaign services
// to persist gameplay effects. For example, DaggerheartService.ApplyRollOutcome
// calls SnapshotService to update character hope/stress and GM fear.
package game
