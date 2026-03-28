// Package detail owns shared campaign workspace-shell transport support.
//
// Route-owned surfaces such as overview, participants, characters, sessions,
// invites, and creation reuse this package for request plumbing, page loading,
// workspace layout, and route-parameter helpers instead of duplicating that
// logic in the campaigns root package.
package detail
