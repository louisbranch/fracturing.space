// Package storage defines persistence contracts for AI aggregates.
//
// The package groups interfaces by workflow or aggregate boundary so callers
// can depend on the smallest repository seam they need while concrete adapters
// remain free to share one runtime root underneath. There is intentionally no
// umbrella repository interface here; callers should depend on the smallest
// capability seam they need. Storage contracts should persist canonical domain
// vocabulary directly when that vocabulary already exists, such as
// `agent.AuthReference` rather than storage-local nullable ID pairs. Support
// packages that own their own workflow vocabulary, such as `providerconnect/`,
// `debugtrace/`, `auditevent/`, and `campaignartifact/`, should also own their
// store seams instead of routing them back through `storage/`.
package storage
