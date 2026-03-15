// Package service wires MCP runtime construction to domain services and
// feature-owned registration packages.
//
// The sibling httptransport package owns the internal HTTP/SSE bridge surface,
// while this package focuses on building MCP server instances, selecting
// registration profiles, and managing gRPC-backed runtime lifecycle. Session
// authority metadata still lives in sessionctx, and focused MCP feature
// packages such as campaigncontext own narrower registration slices.
package service
