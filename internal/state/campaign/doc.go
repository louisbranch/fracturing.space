// Package campaign provides campaign configuration and lifecycle management.
//
// This is the "config layer" of state management - settings that rarely change
// after initial setup: campaign name, game system, GM mode, status, theme prompt.
//
// # Key Types
//
//   - Campaign: The main campaign entity with configuration
//   - CampaignStatus: Lifecycle states (Draft, Active, Completed, Archived)
//   - GmMode: How the GM role is handled (Human, AI, Hybrid)
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
