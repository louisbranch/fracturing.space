// Package server composes application services for the game gRPC entrypoint.
//
// It wires storage, gRPC services, and interceptors into a runnable server
// instance, including event integrity checks for the event journal.
package server
