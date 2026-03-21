// Package rules contains pure game-rule types and functions for the
// Daggerheart system. These definitions depend only on sub-packages
// (profile, contentstore, projectionstore), core/dice, and stdlib —
// they never import the root daggerheart package.
//
// This is the public home for condition, countdown, GM move, damage, and
// adversary rule helpers. Callers that need these deterministic rules should
// depend on `daggerheart/rules` directly rather than the root compatibility
// facade.
package rules
