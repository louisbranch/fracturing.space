// Package campaign defines the campaign aggregate boundary.
//
// Campaign is the strategic root for a game: it owns system selection, GM mode,
// access policy, and lifecycle status. Most other aggregates (session, participants,
// characters, invites) are interpreted relative to that campaign context.
//
// This package exposes:
//   - workflow-local command deciders that translate API intent into immutable campaign events,
//   - folds that rebuild the campaign projection from history,
//   - and policy checks that decide which operations are legal in each lifecycle state.
//
// The one cross-aggregate bootstrap path (`campaign.create_with_participants`)
// intentionally lives in the sibling `campaignbootstrap` workflow package so
// the aggregate decider stays focused on campaign-local rules.
// Aggregate-local decision handling is split by capability so create/update,
// AI, fork, and lifecycle rules stay readable without changing the package
// boundary.
package campaign
