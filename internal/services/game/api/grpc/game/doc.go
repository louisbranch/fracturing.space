// Package game provides system-agnostic gRPC services for game state.
//
// These services manage campaign configuration, participants, invites,
// characters, sessions, snapshots, forks, events, and statistics. They work
// the same regardless of which game system a campaign uses.
//
// Services include:
//   - CampaignService
//   - ParticipantService
//   - InviteService
//   - CharacterService
//   - SessionService
//   - SnapshotService
//   - ForkService
//   - EventService
//   - StatisticsService
package game
