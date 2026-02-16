// Package engine wires command validation, gate checks, decision routing, event
// append, and replay-backed state loading for domain command execution.
//
// Handler is the central orchestrator for command execution. Registries are
// built once at startup, then used to validate and route both core and
// system-owned commands/events.
package engine
