// Package detail owns shared campaign workspace-shell transport support.
//
// Route-owned surfaces such as overview, participants, characters, sessions,
// invites, and creation reuse this package for request plumbing, page loading,
// workspace layout, and route-parameter helpers instead of duplicating that
// logic in the campaigns root package.
//
// # Embedding Depth
//
// Campaign detail handlers have a 5-level embedding chain:
//
//	characters.Handler → detail.Handler → detail.Support → modulehandler.Base → principal.Base
//
// Each level serves a distinct purpose:
//
//   - principal.Base: request-scoped language/viewer resolution (shared with public handlers).
//   - modulehandler.Base: authenticated user resolution, page rendering, error writing,
//     and mutation flash helpers (WriteMutationError, WriteMutationSuccess).
//   - detail.Support: campaign route-param extraction, dashboard sync, time injection,
//     request-scheme policy.
//   - detail.Handler: shared campaign workspace page loading (LoadCampaignPageOrWriteError)
//     and detail-page rendering (WriteCampaignDetailPage).
//   - Surface handlers (characters, sessions, etc.): surface-specific business logic.
//
// When adding methods, place them at the narrowest level that owns the concern.
package detail
