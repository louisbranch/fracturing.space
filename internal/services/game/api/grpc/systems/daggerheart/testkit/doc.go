// Package daggerhearttestkit provides Daggerheart-owned test helpers for
// transport and system package tests that need Daggerheart-specific fixtures.
// Keeping these fakes here avoids leaking system-specific helpers into the
// generic game transport test packages without violating domain import
// boundaries.
package daggerhearttestkit
