// Package storage defines persistence contracts and record types for AI
// aggregates.
//
// The package groups interfaces by workflow or aggregate boundary so callers
// can depend on the smallest repository seam they need while concrete adapters
// remain free to share one runtime root underneath.
package storage
