// Package authz defines the campaign authorization policy matrix.
//
// The package centralizes role/action/resource authorization so transport
// handlers and other services can call one evaluator instead of duplicating
// role checks. Contextual guards (for example ownership or last-owner safety)
// are represented as focused helper evaluators in the same package.
package authz
