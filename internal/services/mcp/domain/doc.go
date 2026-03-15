// Package domain translates MCP UX operations into game domain commands.
//
// The package is intentionally explicit about that mapping:
// - parse MCP request context into game-scoped context,
// - route calls to the correct gRPC domain service,
// - and surface structured outputs that MCP clients can render.
//
// Session-scoped bridge authority, request metadata, and gRPC call plumbing
// live in the sibling sessionctx package so handler code here stays focused on
// MCP-facing gameplay behavior. AI-backed campaign artifact and rules-reference
// handlers live in the sibling campaigncontext package.
//
// This keeps MCP behavior auditable from protocol message -> domain command ->
// projection/read model update.
package domain
