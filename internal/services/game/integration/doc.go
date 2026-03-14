// Package integration defines internal game-to-worker integration contracts.
//
// These payloads and event types are emitted by the game service into its
// integration outbox and consumed by the worker service to perform
// cross-service side effects such as notification intent creation.
//
// The package exists to keep that boundary explicit without reintroducing a
// direct runtime dependency from game write paths to downstream services.
package integration
