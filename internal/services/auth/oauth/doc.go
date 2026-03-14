// Package oauth implements the auth service's first-party OAuth authorization server.
//
// It isolates browser-facing authorization, consent, token, and introspection
// flows from gRPC APIs while keeping identity ownership in auth.
package oauth
