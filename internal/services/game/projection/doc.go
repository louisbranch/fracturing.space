// Package projection builds read models from immutable event history.
//
// Read models are intentionally separate from command aggregates so APIs and UI
// layers can query ergonomic views without loading full aggregate state or
// replaying every event for each request.
//
// Projection is the persistence seam: write-side decisions emit events, projection
// code transforms those events into query-friendly tables and materialized views.
//
// Applier construction is projection-owned: callers bind grouped store concerns
// plus system adapters here instead of rebuilding projection write surfaces in
// transport packages. Core handler registration is also split by concern so
// campaign, participant, invite, session, and scene projection edits do not
// share one mandatory registration file.
// Startup validation is explicit too: core-handler store requirements and
// system-adapter requirements are validated as separate concerns even though
// both feed the same runtime applier.
//
// # Handler Ordering
//
// Projection handlers assume strict event sequence order within a campaign
// journal. Parent entities (Campaign, Session) must be projected before child
// entities (Participant, Character, Scene) because handlers reference parent
// rows via foreign keys and read parent state for denormalized fields.
//
// This ordering is guaranteed by two mechanisms:
//
//  1. The event journal stores events with monotonically increasing sequence
//     numbers per campaign. Storage implementations return events in sequence
//     order from ListEvents.
//
//  2. Replay enforces contiguity: if an event's sequence number doesn't equal
//     lastSeq+1, replay aborts with a gap error rather than silently skipping
//     events that downstream handlers depend on. See [ReplayCampaignWith].
//
// When adding new projection handlers, ensure they tolerate being called for
// events whose parent entity was created in an earlier event within the same
// campaign journal.
package projection
