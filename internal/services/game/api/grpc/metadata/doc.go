// Package metadata provides utilities for handling gRPC request metadata.
//
// It defines standard header keys used across Fracturing.Space services and
// provides interceptors to enforce request IDs and invocation IDs.
//
// Header constants include:
//   - RequestIDHeader: correlates logs and events across service calls
//   - InvocationIDHeader: tracks MCP tool invocations
//   - ParticipantIDHeader/UserIDHeader: identity hints for callers and impersonation
//   - CampaignIDHeader/SessionIDHeader: routing and scoping hints
package metadata
