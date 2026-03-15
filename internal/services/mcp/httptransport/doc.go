// Package httptransport owns the internal MCP streamable-HTTP bridge.
//
// It keeps session lifecycle, request validation, SSE delivery, and per-session
// MCP connection management separate from the service runtime that builds MCP
// registrations and gRPC-backed handlers.
package httptransport
