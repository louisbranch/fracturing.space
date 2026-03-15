// Package workflowtransport owns the shared helper layer for Daggerheart
// gameplay workflows.
//
// This package keeps roll metadata decoding, outcome-code mapping, campaign and
// session metadata propagation, and target normalization out of the root
// Daggerheart transport package so the remaining gameplay handlers can move
// behind a cleaner boundary in later phases.
package workflowtransport
