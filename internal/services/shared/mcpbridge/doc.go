// Package mcpbridge defines the internal MCP bridge contract shared by AI and
// MCP runtimes.
//
// The package exists so production MCP sessions use one explicit source of
// truth for fixed authority and tool exposure instead of duplicating header and
// allowlist rules across services.
package mcpbridge
