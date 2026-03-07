// Package campaigns owns authenticated campaign workspace transport routes.
//
// Type aliases in contracts.go re-export domain types from campaigns/app
// so that handler code uses the root package as its API surface without
// importing app directly. This keeps the module boundary explicit: the
// root package is transport, app holds domain logic and gateway contracts.
// When a new type is added to app/, a corresponding alias is needed here.
package campaigns
