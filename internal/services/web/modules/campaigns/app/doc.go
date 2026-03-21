// Package app contains campaigns domain contracts and orchestration logic.
//
// The campaigns root package owns HTTP transport concerns (module wiring,
// handlers, and routes). This package owns user-scoped validation,
// authorization-aware orchestration, and workflow coordination independent of
// transport.
//
// Capabilities are split by handler surface — catalog, starters, overview,
// participants, characters, creation, sessions, and invites — each with its own
// service interface, config, and gateway contracts. Start with the
// service_contracts_*.go and gateway_contracts_*.go files for the surface you
// need, then follow the config chain back to composition.
package app
