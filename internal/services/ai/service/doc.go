// Package service implements use-case orchestration for the AI service.
//
// Each service struct (AgentService, CredentialService, ProviderGrantService,
// AccessRequestService, InvocationService, CampaignOrchestrationService) owns
// one workflow family. Services accept domain inputs, enforce business rules,
// coordinate storage and cross-service calls, and return domain outputs.
//
// Supporting infrastructure:
//   - AuthTokenResolver resolves agent auth references to live provider tokens
//     (credential decryption or provider-grant refresh).
//   - AccessibleAgentResolver determines whether a user may invoke an agent
//     (ownership or approved access request).
//   - UsageGuard prevents mutations to resources bound to active campaigns.
//   - Error/ErrorKind provides typed service errors that the transport layer
//     maps to gRPC status codes.
//
// The transport layer (api/grpc/ai) is a thin wrapper: parse proto → call
// service → convert result to proto. No business logic lives in transport.
package service
