// Package validator contains Daggerheart payload validation functions used
// by command and event definitions. Each function validates a JSON-encoded
// payload and returns an error when invariants are violated.
//
// The root daggerheart package re-exports selected symbols via function
// variables so root-level test files can reference them by their original
// unexported names.
package validator
