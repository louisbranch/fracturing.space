// Package webctx provides the small request-context bridge used by web
// transport before outbound gRPC calls.
//
// Start here when a protected/public handler needs to enrich request context
// with resolved user metadata. This package should stay minimal and should not
// regrow a second request-state abstraction beside principal/.
package webctx
