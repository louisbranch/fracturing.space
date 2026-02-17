// Package campaign defines the campaign aggregate boundary.
//
// Campaign is the strategic root for a game: it owns system selection, GM mode,
// access policy, and lifecycle status. Most other aggregates (session, participants,
// characters, invites) are interpreted relative to that campaign context.
//
// This package exposes:
//   - command deciders that translate API intent into immutable campaign events,
//   - folds that rebuild the campaign projection from history,
//   - and policy checks that decide which operations are legal in each lifecycle state.
package campaign
