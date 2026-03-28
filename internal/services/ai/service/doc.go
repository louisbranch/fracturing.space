// Package service implements use-case orchestration for the AI service.
//
// Each service struct (AgentService, CredentialService, ProviderGrantService,
// AccessRequestService, InvocationService, CampaignOrchestrationService) owns
// one workflow family. Services accept domain inputs, enforce business rules,
// coordinate storage and cross-service calls, and return domain outputs.
//
// Supporting infrastructure:
//   - AuthMaterialResolver resolves invoke-time auth material for credentials
//     and auth references.
//   - ProviderGrantRuntime owns invoke-time provider-grant refresh policy and
//     lifecycle-state persistence.
//   - AccessibleAgentResolver determines whether a user may invoke an agent
//     (ownership or approved access request).
//   - AgentBindingUsageReader reads active campaign counts for one agent.
//   - AuthReferenceUsageReader maps credential/provider-grant usage through
//     owned agents plus active campaign bindings.
//   - UsagePolicy prevents mutations to resources bound to active campaigns.
//   - Error/ErrorKind provides typed service errors that the transport layer
//     maps to gRPC status codes.
//
// The transport layer (api/grpc/ai) is a thin wrapper: parse proto → call
// service → convert result to proto. No business logic lives in transport.
package service
