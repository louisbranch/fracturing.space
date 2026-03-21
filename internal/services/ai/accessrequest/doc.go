// Package accessrequest owns the access-request domain model and lifecycle.
//
// An access request represents a user's request to invoke another user's agent.
// The lifecycle flows: pending → approved/denied, and approved → revoked.
//
// Domain types (AccessRequest, Page, Scope, Status, Decision) are the single
// representation used by all layers. The storage adapter scans directly into
// domain types; there are no separate storage record types.
//
// Create, Review, and Revoke enforce lifecycle invariants and return the
// transitioned domain object. The service layer in [service] calls these
// functions and persists the result.
package accessrequest
