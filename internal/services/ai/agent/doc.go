// Package agent owns the agent domain model and auth-reference dispatch.
//
// An agent binds an AI provider model to an authentication reference (either a
// stored credential or an OAuth provider grant). The agent lifecycle covers
// creation with provider/model validation, status transitions (active/archived),
// and auth-reference typing.
//
// AuthReference is a sealed sum type dispatched via CredentialID() and
// ProviderGrantID() accessors. Callers determine the auth strategy from these
// without inspecting raw strings.
//
// Domain types (Agent, Page) flow through all layers. The storage adapter scans
// directly into domain types; there are no separate storage record types.
package agent
