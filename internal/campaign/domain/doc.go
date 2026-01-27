// Package domain defines the core business entities and logic for campaign management.
//
// The domain model is centered around several key concepts:
//
// # Campaign
//
// A Campaign represents the top-level entity for a game session. It maintains metadata
// such as the campaign name, GM mode (Human, AI, or Hybrid), and counts for participants
// and characters.
//
// # Participants
//
// Participants are the users involved in a campaign. Each participant has a role
// (GM or Player) and a controller type (Human or AI).
//
// # Characters
//
// Characters represent the entities within the game world. They can be Player Characters (PC)
// or Non-Player Characters (NPC). The character system is split into multiple components:
//   - Character: Core metadata such as name and kind.
//   - CharacterController: Determines if the character is controlled by the GM or a specific Participant.
//   - CharacterProfile: Defines static attributes and maximum values (e.g., HP Max, Traits).
//   - CharacterState: tracks mutable values during gameplay (e.g., current HP, Stress, Hope).
package domain
