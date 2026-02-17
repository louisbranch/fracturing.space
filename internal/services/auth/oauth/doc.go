// Package oauth implements browser-facing OAuth endpoints and external provider flows.
//
// It isolates redirect/state/token choreography from gRPC APIs so the auth
// service can preserve a single identity contract even as provider integrations
// evolve.
package oauth
