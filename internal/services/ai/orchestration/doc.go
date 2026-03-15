// Package orchestration owns the runtime policy for campaign AI turns.
//
// It is responsible for opening MCP sessions with fixed authority, rebuilding
// the provider-facing prompt from authoritative MCP resources, enforcing turn
// safety budgets, and executing the provider/tool loop for one GM turn.
package orchestration
