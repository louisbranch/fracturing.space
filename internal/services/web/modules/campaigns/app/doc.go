// Package app contains campaigns domain contracts and orchestration logic.
//
// The campaigns root package owns HTTP transport concerns (module wiring,
// handlers, and routes). This package owns user-scoped validation,
// authorization-aware orchestration, and workflow coordination independent of
// transport.
package app
