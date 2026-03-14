// Package server composes application services for the game gRPC entrypoint.
//
// The package path is `internal/services/game/app` because it is the runtime
// composition root for the game service. The package name remains `server`
// because callers consume it as the assembled game server surface rather than
// as a generic application helper library.
//
// It wires storage, gRPC services, and interceptors into a runnable server
// instance, including event integrity checks for the event journal.
package server
