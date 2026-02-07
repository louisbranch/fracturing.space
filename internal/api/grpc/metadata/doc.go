// Package metadata provides utilities for handling gRPC request metadata.
//
// It defines standard header keys used by the Duality engine (e.g., Request ID,
// Campaign ID) and provides interceptors to enforce their presence and
// propagation through the system.
//
// # Header Constants
//
//   - RequestIDHeader: Correlates logs and events across service calls.
//   - InvocationIDHeader: Tracks MCP tool invocations.
//   - ParticipantIDHeader/CampaignIDHeader/SessionIDHeader: Contextual hints for routing and identity.
package metadata
