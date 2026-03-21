// Package providergrant owns the provider-grant domain model and lifecycle.
//
// A provider grant represents an OAuth-obtained token pair for a specific user
// and provider. The lifecycle covers creation (from OAuth token exchange),
// token refresh, expiry detection, and revocation.
//
// Key predicates — IsExpired, ShouldRefresh, RefreshSupported, IsUsableBy —
// encapsulate grant health decisions so callers do not inspect raw timestamps
// or status strings.
//
// TokenCiphertext carries the encrypted token data as a domain field.
//
// Domain types (ProviderGrant, Page, Filter) flow through all layers. The
// storage adapter scans directly into domain types; there are no separate
// storage record types.
package providergrant
