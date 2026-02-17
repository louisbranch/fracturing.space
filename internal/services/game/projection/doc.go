// Package projection builds read models from immutable event history.
//
// Read models are intentionally separate from command aggregates so APIs and UI
// layers can query ergonomic views without loading full aggregate state or
// replaying every event for each request.
//
// Projection is the persistence seam: write-side decisions emit events, projection
// code transforms those events into query-friendly tables and materialized views.
package projection
