// Package rules contains pure game-rule types and functions for the
// Daggerheart system. These definitions depend only on sub-packages
// (profile, contentstore, projectionstore), core/dice, and stdlib —
// they never import the root daggerheart package.
//
// The root daggerheart package re-exports all public symbols via type
// aliases and function variables so external consumers are unaffected.
package rules
