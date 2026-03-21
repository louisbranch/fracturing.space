// Package app composes application services for the game gRPC entrypoint.
//
// It is the runtime composition root for the game service: startup sequencing,
// dependency dialing, transport registration, and runnable server lifecycle.
//
// It wires storage, gRPC services, and interceptors into a runnable server
// instance, including event integrity checks for the event journal.
package app
