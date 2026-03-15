// Package creationworkflow implements the Daggerheart side of the shared game
// character-creation workflow contract.
//
// The package owns system-specific creation-step validation, workflow
// application, profile reset/defaulting rules, and workflow error mapping.
// Keeping this provider out of the root Daggerheart gRPC service package makes
// the root package a contribution map for service handlers instead of also
// acting as a cross-package workflow implementation bucket.
package creationworkflow
