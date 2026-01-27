// Package campaign serves as an umbrella for campaign-related functionality,
// including domain models, persistent storage, and gRPC services.
//
// The package is organized into two primary subpackages:
//   - domain: Defines the core business entities (Campaign, Participant, Character)
//     and the rules for their interaction and state management.
//   - service: Implements the gRPC API layer and handles persistence using the
//     underlying storage providers.
package campaign
